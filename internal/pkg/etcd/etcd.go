package etcd

import (
	"context"
	"go-im/internal/pkg/utils"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Config struct {
	Addr string `json:"addr" yaml:"addr"`
	Key  string `json:"key" yaml:"key"`
}

type Client struct {
	cfg *Config
	*clientv3.Client
	leaseID map[string]clientv3.LeaseID
}

func NewClient(cfg Config) *Client {
	if strings.TrimSpace(cfg.Addr) == "" {
		return nil
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.Addr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	return &Client{
		Client:  cli,
		cfg:     &cfg,
		leaseID: make(map[string]clientv3.LeaseID),
	}
}

func (c *Client) Watch(ctx context.Context) clientv3.WatchChan {
	return c.Client.Watch(ctx, c.cfg.Key)
}

func (c *Client) Register(name string, addr string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(addr) == "" {
		panic("etcd target name or addr empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	resp, err := c.Grant(ctx, 60)
	cancel()
	if err != nil {
		return err
	}
	leaseId := resp.ID
	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	_, err = c.Put(ctx, name, addr, clientv3.WithLease(leaseId))
	cancel()
	if err != nil {
		return err
	}
	r, err := c.KeepAlive(context.Background(), leaseId)
	if err != nil {
		return err
	}
	c.leaseID[name] = leaseId
	utils.SafeGo(func() {
		for {
			<-r
		}
	})
	return nil
}

func (c *Client) UnRegister(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	c.Delete(ctx, name)
	cancel()

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	c.Revoke(ctx, c.leaseID[name])
	delete(c.leaseID, name)
	cancel()
}
