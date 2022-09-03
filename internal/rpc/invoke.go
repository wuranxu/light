package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shimingyah/pool"
	"github.com/wuranxu/light/internal/auth"
	"github.com/wuranxu/light/internal/service/etcd"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	//cc  *grpc.ClientConn
	cli *etcd.Client
	po  pool.Pool
}

func (c *GrpcClient) Invoke(method etcd.Method, in *Request, ip string, userInfo *auth.UserInfo, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	md := metadata.New(map[string]string{"host": ip})
	if userInfo != nil {
		md.Append("user", userInfo.Marshal())
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	cn, err := c.po.Get()
	if err != nil {
		return out, err
	}
	if err := cn.Value().Invoke(ctx, method.Path, in, out, opts...); err != nil {
		return out, err
	}
	return out, nil
}

func (c *GrpcClient) Close() error {
	if c != nil {
		return c.po.Close()
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
	p, err := pool.New(service, pool.Options{
		Dial: func(address string) (*grpc.ClientConn, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			addr := fmt.Sprintf("%s:///%s", etcd.Resolver.Scheme(), service)
			return grpc.DialContext(ctx, addr,
				grpc.WithBlock(),
				grpc.WithReturnConnectionError(),
				grpc.WithDefaultServiceConfig(invokeConfig), grpc.WithInsecure())
		},
		MaxIdle:              8,
		MaxActive:            64,
		MaxConcurrentStreams: 64,
		Reuse:                true,
	})
	if err != nil {
		return nil, err
	}
	return &GrpcClient{etcd.Cli, p}, nil
}
