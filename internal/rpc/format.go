package rpc

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"strings"
	"sync"
)

type anyResolver struct {
	source DescriptorSource

	er dynamic.ExtensionRegistry

	mu       sync.RWMutex
	mf       *dynamic.MessageFactory
	resolved map[string]func() proto.Message
}

func (r *anyResolver) Resolve(typeUrl string) (proto.Message, error) {
	mname := typeUrl
	if slash := strings.LastIndex(mname, "/"); slash >= 0 {
		mname = mname[slash+1:]
	}

	r.mu.RLock()
	factory := r.resolved[mname]
	r.mu.RUnlock()

	// already resolved?
	if factory != nil {
		return factory(), nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// double-check, in case we were racing with another goroutine
	// that resolved this one
	factory = r.resolved[mname]
	if factory != nil {
		return factory(), nil
	}

	// use descriptor source to resolve message type
	d, err := r.source.FindSymbol(mname)
	if err != nil {
		return nil, err
	}
	md, ok := d.(*desc.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("unknown message: %s", typeUrl)
	}
	// populate any extensions for this message, too
	if exts, err := r.source.AllExtensionsForType(mname); err != nil {
		return nil, err
	} else if err := r.er.AddExtension(exts...); err != nil {
		return nil, err
	}

	if r.mf == nil {
		r.mf = dynamic.NewMessageFactoryWithExtensionRegistry(&r.er)
	}

	factory = func() proto.Message {
		return r.mf.NewMessage(md)
	}
	if r.resolved == nil {
		r.resolved = map[string]func() proto.Message{}
	}
	r.resolved[mname] = factory
	return factory(), nil
}
