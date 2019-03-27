package main

import (
	"context"
	"fmt"
)

// RunController - Run the loop to periodically write the loop
func RunController(ctx context.Context, lease LeaseInf, inf InformerInf) {

	//Todo: Read kubernetes master for the master nodes

	// etcdctl put /skydns/local/skydns/x1 '{"host":"1.1.1.1","ttl":60}' - on the first node
	// etcdctl put /skydns/local/skydns/x2 '{"host":"1.1.1.2","ttl":60}' - on the second node
loop:
	for {

		//Initally connect to etcd server and get the interupt channel
		errLease := lease.StartLease(ctx, "key1", "val1")
		if errLease == nil {
			select {
			case <-lease.GetRenewalInteruptChan():
				fmt.Println("Controller detected a interuption on the renewal")
				//Do nothing

			case <-inf.GetInformerInterupt():

				err := lease.RevokeLease(ctx)
				if err != nil {
					//What is handling here... I am not sure yet
				}
			case <-ctx.Done():
				break loop

			}
		}
	}

}

// LeaseInf - Enable the controller to start leasing and wait for the signal to change flow
type LeaseInf interface {
	StartLease(ctx context.Context, kvKey string, kvVal string) error
	GetRenewalInteruptChan() (renewalInterupted chan struct{})
	RevokeLease(ctx context.Context) error
}

// InformerInf - Enable the controller to determine what value to be writen to the Lease
type InformerInf interface {
	GetDnsKeyVal(ctx context.Context) (dnsname string, host string, ttl string, err error)
	GetInformerInterupt() (informerInterupted chan struct{})
}
