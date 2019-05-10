package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
)

// Entry - Key Val entry for dns entry
type Entry struct {
	Key string
	Val string
}

// EtcdLease - Managed all the etcd connection and renewal
type EtcdLease struct {
	//cl                *clientv3.Client
	lease             clientv3.Lease
	leaseID           clientv3.LeaseID
	renewalInterupted chan struct{}

	client *clientv3.Client

	//Deadline for the leasing, this when the channel would set to nil
	leaseTimeInSec int

	//This to check the channel is still healthy
	renewTickCheck int
}

// NewEtcdLease - Establish a new Lease for next op
func NewEtcdLease(client *clientv3.Client) *EtcdLease {

	//% etcdctl put /skydns/local/skydns/x1 '{"host":"1.1.1.1","ttl":60}'
	//% etcdctl put /skydns/local/skydns/x2 '{"host":"1.1.1.2","ttl":60}'

	return &EtcdLease{
		client: client,
		// leaseTimeInSec: leaseTimeInSec,
		// renewTickCheck: renewTickCheck,
	}
}

// LeaseStatus - Meant to use for troubleshooting
func (e *EtcdLease) LeaseStatus() {
	ttlresp, err := e.lease.TimeToLive(context.Background(), e.leaseID)
	if err != nil {
		fmt.Println(fmt.Errorf(err.Error()))
		return
	}
	fmt.Println(ttlresp)
}

// StartLease - Start watching for lease for the key and value
func (e *EtcdLease) StartLease(ctx context.Context, entries []Entry, leaseTimeInSec int) error {

	err := e.InitLease(ctx, entries, leaseTimeInSec)
	if err != nil {
		return err
	}

	e.renewalInterupted, err = e.RenewLease(ctx)
	if err != nil {
		return err
	}

	return nil
}

// InitLease - Initalize Lease
// Ideally leaseTimeInSec should be less than renewTickCheck
func (e *EtcdLease) InitLease(ctx context.Context, entries []Entry, leaseTimeInSec int) error {
	e.UpdateKeepAliveTime(leaseTimeInSec)

	kv := clientv3.NewKV(e.client)
	e.lease = clientv3.NewLease(e.client)

	//Grant the lease with the time
	leaseResp, err := e.lease.Grant(ctx, int64(e.leaseTimeInSec))
	if err != nil {
		fmt.Println("Could not setup the lease " + err.Error())
		return err
	}

	e.leaseID = leaseResp.ID

	//Write a list of entries into etcd
	for _, entry := range entries {
		//Attemp to write the key value into etcd with the lease
		_, err = kv.Put(ctx, entry.Key, entry.Val, clientv3.WithLease(e.leaseID))
		if err != nil {
			fmt.Println("Could not write to store: " + err.Error())
			return err
		}
	}

	return nil

}

// UpdateKeepAliveTime - It is not practical to used the same LeaseTime as DNS TTL Time
func (e *EtcdLease) UpdateKeepAliveTime(leaseTimeInSec int) {

	e.leaseTimeInSec = leaseTimeInSec

	if leaseTimeInSec <= 3 {
		e.leaseTimeInSec = 1
	}

	e.renewTickCheck = leaseTimeInSec / 2

}

// RenewLease - Keep on renewing the lease
func (e *EtcdLease) RenewLease(ctx context.Context) (renewalInterupted chan struct{}, err error) {

	//Keep the lease alive forever
	kaCh, err := e.lease.KeepAlive(ctx, e.leaseID)

	if err != nil {
		fmt.Println("Could not renew lease " + err.Error())
		return
	}

	//Error before this become a panic
	if e.renewTickCheck <= 0 {
		return nil, errors.New("Can not use non value")
	}

	renewalTicker := time.NewTicker(time.Duration(e.renewTickCheck) * time.Second)

	renewalInterupted = make(chan struct{})

	//Run a separate goroutine to check if the renewal is interupted, otherwise indicate to parent the renewal is interupted
	go func() {

	loop:
		for kaCh != nil {

			select {
			case _, ok := <-kaCh:
				fmt.Println("Refreshing channel")
				if !ok {
					fmt.Println("Channel not ok")
					break loop
				}
				// else {
				// 	fmt.Println(karesp.TTL)
				// }
			case <-renewalTicker.C:
				fmt.Println("Rechecking keep alive channel is closed")
			case <-ctx.Done(): //If a parent context requested to be closed
				fmt.Println("Closing RenewLease routine:" + fmt.Sprintf("%d", e.leaseID))
				return //Terminate the go routine
			}

		}

		renewalTicker.Stop() //Stop timer
		fmt.Println("Signaled Interuption")
		renewalInterupted <- struct{}{}
		return
	}()

	return
}

// GetRenewalInteruptChan - Give the caller to redirect the program flow due to interuption
func (e *EtcdLease) GetRenewalInteruptChan() (renewalInterupted chan struct{}) {
	return e.renewalInterupted
}

// RevokeLease -
func (e *EtcdLease) RevokeLease(ctx context.Context) error {
	_, err := e.client.Revoke(ctx, e.leaseID)
	return err

}
