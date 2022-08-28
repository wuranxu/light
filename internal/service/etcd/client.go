package etcd

import (
	"context"
	"github.com/wuranxu/light/conf"
	v3 "go.etcd.io/etcd/client/v3"
	re "google.golang.org/grpc/resolver"
	"time"
)

type Client struct {
	kv     v3.KV
	cli    *v3.Client
	scheme string
}

var (
	Cli      *Client
	Resolver re.Builder
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

func Init(cfg conf.EtcdConfig) error {
	var err error
	cli, err := v3.New(v3.Config{Endpoints: cfg.Endpoints, DialTimeout: time.Second * time.Duration(cfg.DialTimeout)})
	if err != nil {
		return err
	}
	//timeout, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	//defer cancelFunc()
	//if _, err = cli.Authenticate(timeout, cfg.Username, cfg.Password); err != nil {
	//	return err
	//}
	kv := v3.NewKV(cli)
	Cli = &Client{kv: kv, cli: cli, scheme: cfg.Scheme}
	Resolver = NewResolver(Cli, conf.Conf.Etcd.Scheme)
	return nil

}
