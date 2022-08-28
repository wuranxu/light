package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wuranxu/light/internal/auth"
	"github.com/wuranxu/light/internal/service/etcd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
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
}

func (c *GrpcClient) Invoke(method etcd.Method, in *Request, ip string, userInfo *auth.CustomClaims, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	md := metadata.New(map[string]string{"host": ip})
	if userInfo != nil {
		md.Append("user", userInfo.Marshal())
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	if err := c.cc.Invoke(ctx, method.Path, in, out, opts...); err != nil {
		return out, err
	}
	return out, nil
}

func (c *GrpcClient) Close() error {
	if c != nil {
		return c.cc.Close()
	}
	return nil
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
	resolver.Register(etcd.Resolver)
	conn, err := grpc.DialContext(context.Background(), fmt.Sprintf("%s:///%s", etcd.Resolver.Scheme(), service),
		grpc.WithDefaultServiceConfig(invokeConfig), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &GrpcClient{cli: etcd.Cli, cc: conn}, nil
}
