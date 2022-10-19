package rpc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/wuranxu/light/internal/auth"
	"github.com/wuranxu/light/internal/service/etcd"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"io"
	"strings"
	"time"
)

var (
	MethodNotFound = errors.New("没有找到对应的方法，请检查您的参数")
	invokeConfig   = `{
	  "loadBalancingConfig": [ { "round_robin": {} } ],
	  "methodConfig": []
	}
	`
)

type GrpcClient struct {
	cc  *grpc.ClientConn
	cli *etcd.Client
	rc  *ReflectionClient
}

//func (c *GrpcClient) Invoke(method etcd.Method, in *Request, ip string, userInfo *auth.UserInfo, opts ...grpc.CallOption) (*Response, error) {
//	out := new(Response)
//	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
//	defer cancel()
//	md := metadata.New(map[string]string{"host": ip})
//	if userInfo != nil {
//		md.Append("user", userInfo.Marshal())
//	}
//	ctx = metadata.NewOutgoingContext(ctx, md)
//	if err := c.cc.Invoke(ctx, method.Path, in, out, opts...); err != nil {
//		return out, err
//	}
//	return out, nil
//}

func (c *GrpcClient) InvokeWithReflect(method etcd.Method, in io.ReadCloser, ip string, userInfo *auth.UserInfo, opts ...grpc.CallOption) (proto.Message, error) {
	split := strings.Split(method.Path, "/")
	service, mth := split[len(split)-2], split[len(split)-1]
	md := metadata.New(map[string]string{"host": ip})
	if userInfo != nil {
		md.Append("user", base64.StdEncoding.EncodeToString(userInfo.Marshal()))
	}
	client := c.rc
	cache, err := client.Args(service, mth, in)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	ctx = metadata.NewOutgoingContext(ctx, md)
	defer cancel()
	err = c.cc.Invoke(ctx, method.Path, cache.req, cache.res, opts...)
	//unary, err := client.InvokeUnary(ctx, cache.msgFactory, cache.md, cache.req, opts...)
	//fmt.Println(time.Now().Unix())
	//return unary, err
	return cache.res, err
}

func (c *GrpcClient) Marshal(w io.Writer, msg proto.Message) error {
	return c.rc.Marshal(w, msg)
}

func (c *GrpcClient) Close() error {
	//if c != nil {
	//	return c.po.Close()
	//}
	//return nil
	return c.cc.Close()
}

func (c *GrpcClient) SearchCallAddr(version, service, method string) (etcd.Method, error) {
	var md etcd.Method
	addr := c.cli.GetSingle(fmt.Sprintf("%s.%s.%s", version, service, method))
	if addr == "" {
		//log.E("版本:[%s] 服务:[%s] 方法:[%s]未找到", version, service, method)
		return md, MethodNotFound
	}
	if err := json.Unmarshal([]byte(addr), &md); err != nil {
		return md, err
	}
	return md, nil
}

func NewGrpcClient(service string) (*GrpcClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	//addr := fmt.Sprintf("%s:///%s", etcd.Resolver.Scheme(), service)
	etcdResolver, err := resolver.NewBuilder(etcd.Cli.Cli())
	if err != nil {
		return nil, err
	}
	conn, err := grpc.DialContext(ctx, "etcd:///"+service, grpc.WithResolvers(etcdResolver), grpc.WithInsecure())
	//conn, err := grpc.DialContext(ctx, addr,
	//	grpc.WithResolvers(etcd.Resolver),
	//	grpc.WithDefaultServiceConfig(invokeConfig),
	//	grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &GrpcClient{conn, etcd.Cli, NewReflectionClient(conn)}, nil
}
