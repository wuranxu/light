package etcd

import (
	"context"
	"github.com/wuranxu/light/conf"
	v3 "go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

type Client struct {
	kv     v3.KV
	cli    *v3.Client
	scheme string
}

var (
	once       sync.Once
	EtcdClient *Client
)

func (cl *Client) Kv() v3.KV {
	return cl.kv
}

func (cl *Client) Cli() *v3.Client {
	return cl.cli
}

func (cl *Client) Set(key, value string) bool {
	_, err := cl.kv.Put(context.TODO(), key, value, v3.WithPrevKV())
	if err != nil {
		return false
	}
	return true
}

func (cl *Client) GetSingle(key string) string {
	res, err := cl.kv.Get(context.TODO(), key)
	if err != nil {
		return ""
	}
	if len(res.Kvs) == 0 {
		return ""
	}
	return string(res.Kvs[0].Value)
}

func (cl *Client) GetPattern(key string) (result map[string]string) {
	res, err := cl.kv.Get(context.TODO(), key, v3.WithPrefix())
	result = make(map[string]string)
	if err != nil {
		return
	}
	for _, item := range res.Kvs {
		result[string(item.Key)] = string(item.Value)
	}
	return
}

func (cl *Client) Close() error {
	return cl.cli.Close()
}

func NewClient(cfg conf.EtcdConfig) (*Client, error) {
	var (
		err error
		cli *v3.Client
		kv  v3.KV
	)
	if EtcdClient == nil {
		once.Do(func() {
			cli, err = v3.New(v3.Config{Endpoints: cfg.Endpoints, DialTimeout: time.Second * cfg.DialTimeout})
			if err != nil {
				cli = nil
			}
			kv = v3.NewKV(cli)

		})
		EtcdClient = &Client{kv: kv, cli: cli, scheme: cfg.Scheme}
	}
	if err != nil {
		return nil, err
	}
	return EtcdClient, nil
}
