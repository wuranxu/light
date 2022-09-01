package etcd

import (
	"context"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	re "google.golang.org/grpc/resolver"
	"strings"
	"sync"
)

type resolver struct {
	scheme  string
	client  *Client
	cc      re.ClientConn
	lock    sync.RWMutex
	watcher map[string]bool
}

func NewResolver(client *Client, scheme string) re.Builder {
	return &resolver{client: client, scheme: scheme, watcher: make(map[string]bool)}
}

func (r *resolver) Get(key string) bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	_, ok := r.watcher[key]
	return ok
}

func (r *resolver) Set(key string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.watcher[key] = true
}

func (r *resolver) Build(target re.Target, cc re.ClientConn, opt re.BuildOptions) (re.Resolver, error) {
	r.cc = cc
	key := "/" + target.Scheme + "/" + target.Endpoint + "/"
	address := r.addr(key)
	if !r.Get(key) {
		go r.watch(key, &address)
		r.Set(key)
	}
	return r, nil
}

func (r *resolver) Scheme() string {
	return r.scheme
}

func (r *resolver) ResolveNow(rn re.ResolveNowOptions) {

}

func (r *resolver) Close() {

}

func (r *resolver) addr(keyPrefix string) []re.Address {
	var addrList []re.Address
	getResp, err := r.client.cli.Get(context.Background(), keyPrefix, clientv3.WithPrefix())
	if err != nil {
		// TODO
		return nil
	}
	for i := range getResp.Kvs {
		addrList = append(addrList, re.Address{Addr: strings.TrimPrefix(string(getResp.Kvs[i].Key), keyPrefix)})
	}
	state := re.State{Addresses: addrList}
	r.cc.UpdateState(state)
	return addrList
}

func (r *resolver) watch(keyPrefix string, addrList *[]re.Address) {
	rch := r.client.cli.Watch(context.Background(), keyPrefix, clientv3.WithPrefix())
	for n := range rch {
		for _, ev := range n.Events {
			addr := strings.TrimPrefix(string(ev.Kv.Key), keyPrefix)
			switch ev.Type {
			case mvccpb.PUT:
				if !exist(*addrList, addr) {
					*addrList = append(*addrList, re.Address{Addr: addr})
				}
			case mvccpb.DELETE:
				if s, ok := remove(*addrList, addr); ok {
					addrList = &s
					r.cc.UpdateState(re.State{Addresses: *addrList})
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
