package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/wuranxu/light/internal/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"io"
	"sync"
)

var (
	MemCache = &MemoryCache{cache: make(map[string]*MethodCache)}
)

type ReflectionClient struct {
	conn       *grpc.ClientConn
	client     *grpcreflect.Client
	descSource DescriptorSource
	dec        *json.Decoder
}

func (r *ReflectionClient) Marshal(w io.Writer, msg proto.Message) error {
	resolver := &anyResolver{source: r.descSource}
	m := jsonpb.Marshaler{AnyResolver: resolver, EmitDefaults: true, Indent: "    "}
	return m.Marshal(w, msg)
}

type MemoryCache struct {
	lock  sync.RWMutex
	cache map[string]*MethodCache
}

type MethodCache struct {
	msgFactory *dynamic.MessageFactory
	src        *desc.ServiceDescriptor
	md         *desc.MethodDescriptor
	req        proto.Message
	res        proto.Message
}

func (r *MemoryCache) GetCache(service, method string) *MethodCache {
	r.lock.RLock()
	defer r.lock.RUnlock()
	c, ok := r.cache[service+"/"+method]
	if !ok {
		return nil
	}
	return c
}

func (r *MemoryCache) SetCache(service, method string, cache *MethodCache) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cache[service+"/"+method] = cache
}

func DescriptorSourceFromServer(_ context.Context, refClient *grpcreflect.Client) DescriptorSource {
	return serverSource{client: refClient}
}

func NewReflectionClient(conn *grpc.ClientConn) *ReflectionClient {
	ctx := context.Background()
	refClient := grpcreflect.NewClient(ctx, reflectpb.NewServerReflectionClient(conn))
	reflSource := DescriptorSourceFromServer(ctx, refClient)
	return &ReflectionClient{descSource: reflSource, conn: conn}
}

func (r *ReflectionClient) FindSymbol(service string) (desc.Descriptor, error) {
	dsc, err := r.descSource.FindSymbol(service)
	if err != nil {
		errStatus, hasStatus := status.FromError(err)
		switch {
		case hasStatus && errors.IsNotFoundError(err):
			return nil, status.Errorf(errStatus.Code(), "target server does not expose service %q: %s", service, errStatus.Message())
		case hasStatus:
			return nil, status.Errorf(errStatus.Code(), "failed to query for service descriptor %q: %s", service, errStatus.Message())
		case errors.IsNotFoundError(err):
			return nil, fmt.Errorf("target server does not expose service %q", service)
		}
		return nil, fmt.Errorf("failed to query for service descriptor %q: %v", service, err)
	}
	return dsc, nil
}

func (r *ReflectionClient) Descriptor(dsc desc.Descriptor, service, method string) (*MethodCache, error) {
	sd, ok := dsc.(*desc.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("target server does not expose service %q", service)
	}
	mtd := sd.FindMethodByName(method)
	if mtd == nil {
		return nil, fmt.Errorf("service %q does not include a method named %q", service, method)
	}
	var ext dynamic.ExtensionRegistry
	alreadyFetched := map[string]bool{}
	if err := fetchAllExtensions(r.descSource, &ext, mtd.GetInputType(), alreadyFetched); err != nil {
		return nil, fmt.Errorf("error resolving server extensions for message %s: %v", mtd.GetInputType().GetFullyQualifiedName(), err)
	}
	if err := fetchAllExtensions(r.descSource, &ext, mtd.GetOutputType(), alreadyFetched); err != nil {
		return nil, fmt.Errorf("error resolving server extensions for message %s: %v", mtd.GetOutputType().GetFullyQualifiedName(), err)
	}
	msgFactory := dynamic.NewMessageFactoryWithExtensionRegistry(&ext)
	req := msgFactory.NewMessage(mtd.GetInputType())
	res := msgFactory.NewMessage(mtd.GetOutputType())
	return &MethodCache{
		src:        sd,
		md:         mtd,
		req:        req,
		res:        res,
		msgFactory: msgFactory,
	}, nil
}

func (r *ReflectionClient) Stub(msgFactory *dynamic.MessageFactory) grpcdynamic.Stub {
	return grpcdynamic.NewStubWithMessageFactory(r.conn, msgFactory)

}

func (r *ReflectionClient) Args(service, method string, in io.ReadCloser) (*MethodCache, error) {
	//cache := MemCache.GetCache(service, method)
	//if cache == nil {
	//	dsc, err := r.FindSymbol(service)
	//	if err != nil {
	//		return nil, err
	//	}
	//	cache, err = r.Descriptor(dsc, service, method)
	//	if err != nil {
	//		return nil, err
	//	}
	//	MemCache.SetCache(service, method, cache)
	//}
	dsc, err := r.FindSymbol(service)
	if err != nil {
		return nil, err
	}
	cache, err := r.Descriptor(dsc, service, method)
	if err != nil {
		return nil, err
	}

	var msg json.RawMessage
	dec := json.NewDecoder(in)
	if err := dec.Decode(&msg); err != nil {
		return nil, err
	}
	resolver := &anyResolver{source: r.descSource}
	unmarshaler := jsonpb.Unmarshaler{AnyResolver: resolver, AllowUnknownFields: true}
	err = unmarshaler.Unmarshal(bytes.NewReader(msg), cache.req)
	return cache, err
}

func (r *ReflectionClient) InvokeUnary(ctx context.Context, msgFactory *dynamic.MessageFactory, method *desc.MethodDescriptor, req proto.Message, opts ...grpc.CallOption) (proto.Message, error) {
	// Now we can actually invoke the RPC!
	var respHeaders metadata.MD
	var respTrailers metadata.MD
	options := []grpc.CallOption{grpc.Trailer(&respTrailers), grpc.Header(&respHeaders)}
	if opts != nil {
		options = append(options, opts...)
	}
	stub := r.Stub(msgFactory)
	resp, err := stub.InvokeRpc(ctx, method, req, options...)

	stat, ok := status.FromError(err)
	if !ok {
		// Error codes sent from the server will get printed differently below.
		// So just bail for other kinds of errors here.
		return nil, fmt.Errorf("grpc call for %q failed: %v", method.GetFullyQualifiedName(), err)
	}
	if stat.Code() == codes.OK {
		return resp, stat.Err()
	}
	return resp, nil
}

func fetchAllExtensions(source DescriptorSource, ext *dynamic.ExtensionRegistry, md *desc.MessageDescriptor, alreadyFetched map[string]bool) error {
	msgTypeName := md.GetFullyQualifiedName()
	if alreadyFetched[msgTypeName] {
		return nil
	}
	alreadyFetched[msgTypeName] = true
	if len(md.GetExtensionRanges()) > 0 {
		fds, err := source.AllExtensionsForType(msgTypeName)
		if err != nil {
			return fmt.Errorf("failed to query for extensions of type %s: %v", msgTypeName, err)
		}
		for _, fd := range fds {
			if err := ext.AddExtension(fd); err != nil {
				return fmt.Errorf("could not register extension %s of type %s: %v", fd.GetFullyQualifiedName(), msgTypeName, err)
			}
		}
	}
	// recursively fetch extensions for the types of any message fields
	for _, fd := range md.GetFields() {
		if fd.GetMessageType() != nil {
			err := fetchAllExtensions(source, ext, fd.GetMessageType(), alreadyFetched)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
