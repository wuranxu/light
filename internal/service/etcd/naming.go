package etcd

import (
	"context"
	"github.com/wuranxu/light/conf"
	"go.etcd.io/etcd/client/v3"
	"log"
	"reflect"
	"time"
)

func (cl *Client) RegisterService(name, addr string, ttl int64) error {
	//ticker := time.NewTicker(time.Second * time.Duration(ttl))

	go func() {
		for {
			getResp, err := cl.cli.Get(context.Background(), "/"+cl.scheme+"/"+name+"/"+addr)
			if err != nil {
				log.Printf("获取服务信息失败, error: %s", err)
			} else if getResp.Count == 0 {
				if err = cl.withAlive(name, addr, ttl); err != nil {
					log.Fatalf("注册服务失败, error: %s", err)
				}
			} else {
			}
			//<-ticker.C
			time.Sleep(time.Second * time.Duration(ttl))
		}
	}()
	return nil
}

func (cl *Client) withAlive(name, addr string, ttl int64) error {
	leaseResp, err := cl.cli.Grant(context.Background(), ttl)
	if err != nil {
		return err
	}
	log.Printf("service alive:%v\n", "/"+cl.scheme+"/"+name+"/"+addr)
	if _, err := cl.cli.Put(context.Background(), "/"+cl.scheme+"/"+name+"/"+addr, addr, clientv3.WithLease(leaseResp.ID)); err != nil {
		return err
	}

	if _, err := cl.cli.KeepAlive(context.Background(), leaseResp.ID); err != nil {
		return err
	}
	return nil
}

func (cl *Client) UnRegister(name, addr string) error {
	if cl.cli != nil {
		_, err := cl.cli.Delete(context.Background(), "/"+cl.scheme+"/"+name+"/"+addr)
		return err
	}
	return nil
}

func (cl *Client) RegisterApi(name string, data interface{}, config conf.YamlConfig) error {
	inf := reflect.ValueOf(data)
	for i := 0; i < inf.NumMethod(); i++ {
		methodName := inf.Type().Method(i).Name
		md, ok := config.Method[methodName]
		if !ok {
			// 说明配置文件没有包含此方法
			log.Fatal("注册Api失败, service.yaml文件未包含此方法: ", methodName)
		}
		err := RegisterMethod(cl, config.Version, name, methodName, md.Authorization)
		if err != nil {
			return err
		}
	}
	return nil
}

// 注销方法
func (cl *Client) UnRegisterApi(name string, data interface{}, config conf.YamlConfig) error {
	inf := reflect.ValueOf(data)
	for i := 0; i < inf.NumMethod(); i++ {
		methodName := inf.Type().Method(i).Name
		_, ok := config.Method[methodName]
		if !ok {
			// 说明配置文件没有包含此方法
			log.Fatal("注册Api失败, service.yaml文件未包含此方法: ", methodName)
		}
		err := UnRegisterMethod(cl, config.Version, name, methodName)
		if err != nil {
			return err
		}
	}
	return nil
}
