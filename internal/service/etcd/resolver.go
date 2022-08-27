package etcd

import (
	"context"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	re "google.golang.org/grpc/resolver"
	"strings"
)

type resolver struct {
	scheme string
	client *Client
	cc     re.ClientConn
}

func NewResolver(client *Client, scheme string) re.Builder {
	return &resolver{client: client, scheme: scheme}
}

func (r *resolver) Build(target re.Target, cc re.ClientConn, opt re.BuildOptions) (re.Resolver, error) {
	r.cc = cc
	go r.watch("/" + target.Scheme + "/" + target.Endpoint + "/")
	return r, nil
}

func (r *resolver) Scheme() string {
	return r.scheme
}

func (r *resolver) ResolveNow(rn re.ResolveNowOptions) {

}

func (r *resolver) Close() {

}

func (r *resolver) watch(keyPrefix string) {
	var addrList []re.Address
	getResp, err := r.client.cli.Get(context.Background(), keyPrefix, clientv3.WithPrefix())
	if err != nil {
		// TODO
		return
	}
	for i := range getResp.Kvs {
		addrList = append(addrList, re.Address{Addr: strings.TrimPrefix(string(getResp.Kvs[i].Key), keyPrefix)})
	}
	state := re.State{Addresses: addrList}
	r.cc.UpdateState(state)

	rch := r.client.cli.Watch(context.Background(), keyPrefix, clientv3.WithPrefix())
	for n := range rch {
		for _, ev := range n.Events {
			addr := strings.TrimPrefix(string(ev.Kv.Key), keyPrefix)
			switch ev.Type {
			case mvccpb.PUT:
				if !exist(addrList, addr) {
					addrList = append(addrList, re.Address{Addr: addr})
				}
			case mvccpb.DELETE:
				if s, ok := remove(addrList, addr); ok {
					addrList = s
					r.cc.UpdateState(re.State{Addresses: addrList})
				}

			}
		}
	}

}

func exist(l []re.Address, addr string) bool {
	for _, item := range l {
		if item.Addr == addr {
			return true
		}
	}
	return false
}

func remove(l []re.Address, addr string) ([]re.Address, bool) {
	for i, item := range l {
		if item.Addr == addr {
			l[i] = l[len(l)-1]
			return l[:len(l)-1], true
		}
	}
	return nil, false
}
