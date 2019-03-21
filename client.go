package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc"
)

// Testing
func FuncA() bool {
	// expect dial time-out on ipv4 blackhole
	cl, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://127.0.0.1:2378"},
		DialTimeout: 2 * time.Second,
	})
	defer cl.Close()

	// etcd clientv3 >= v3.2.10, grpc/grpc-go >= v1.7.3
	if err == context.DeadlineExceeded {
		return false
	}

	// etcd clientv3 <= v3.2.9, grpc/grpc-go <= v1.2.1
	if err == grpc.ErrClientConnTimeout {
		return false
	}

	timeout := 2 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	_, err = cl.Put(ctx, "abc", "def")

	defer cancel()

	if err != nil {

		return false
	}

	resp, err := cl.Get(ctx, "abc")
	fmt.Println(resp.Kvs)

	return true

}
