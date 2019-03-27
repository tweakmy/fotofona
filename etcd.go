package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc"
)

// NewGrant - Prepare a new grant for the enviroment to use
func NewGrant(ctx context.Context, endpoint string, leaseTimeInSec int64,
	kvKey string, kvVal string) (clientv3.KV, clientv3.LeaseID, clientv3.Lease, error) {

	//Create New Client
	cl, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 2 * time.Second,
	})

	if err != nil {
		fmt.Println("Fail to connect to etcd server " + err.Error())
		return nil, 0, nil, err
	}

	kv := clientv3.NewKV(cl)
	lease := clientv3.NewLease(cl)

	//Grant the lease with the time
	leaseResp, err := lease.Grant(ctx, leaseTimeInSec)
	if err != nil {
		fmt.Println("Could not setup the lease " + err.Error())
		return nil, 0, nil, err
	}

	//Attemp to write the key value into etcd with the lease
	_, err = kv.Put(ctx, kvKey, kvVal, clientv3.WithLease(leaseResp.ID))
	if err != nil {
		fmt.Println("Could not write to store: " + err.Error())
		return nil, 0, nil, err
	}

	return kv, leaseResp.ID, lease, nil

}

// RenewLeaseOnce - Renewing lease once
func RenewLeaseOnce(ctx context.Context, lease clientv3.Lease, leaseID clientv3.LeaseID) (err error) {

	_, err = lease.KeepAliveOnce(ctx, leaseID)
	if err != nil {
		fmt.Println("Could not renew lease once " + err.Error())
	}

	return
}

// RenewLease - Keep on renewing the lease
func RenewLease(ctx context.Context, lease clientv3.Lease, leaseID clientv3.LeaseID,
	timeinsec int64) (renewalInterupted chan struct{}, err error) {

	kaCh, err := lease.KeepAlive(ctx, leaseID)

	if err != nil {
		fmt.Println("Could not renew lease " + err.Error())
		return
	}

	renewalTicker := time.NewTicker(time.Duration(timeinsec) * time.Second)

	renewalInterupted = make(chan struct{})

	//Run a separate goroutine to check if the renewal is interupted, otherwise indicate to parent the renewal is interupted
	go func() {

	loop:
		for kaCh != nil {

			select {
			case _, ok := <-kaCh:
				if !ok {
					break loop
				}
				// else {
				// 	fmt.Println(karesp.TTL)
				// }
			case <-renewalTicker.C:
				fmt.Println("Channel is not closed")
			}

		}

		renewalTicker.Stop() //Stop timer
		renewalInterupted <- struct{}{}
		return
	}()

	return
}

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
