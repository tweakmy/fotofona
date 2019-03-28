package main

import (
	"context"
	"fmt"
	"strings"
)

// RunController - Run the loop to periodically write the loop
func RunController(ctx context.Context, rootKey string, dnsname string, lease LeaseInf, inf InformerInf) {

	//Dns name remain constant over long period of time
	dnsArry := reverseArray(strings.Split(dnsname, "."))

loop:
	for {

		var errLease error
		var key1, val1 string

		hostip, ttl, uniqueNumber, err := inf.GetDNSKeyVal(ctx)
		if err != nil {
			goto retry
		}

		// etcdctl put /skydns/local/skydns/x1 '{"host":"1.1.1.1","ttl":60}' - on the first node
		// etcdctl put /skydns/local/skydns/x2 '{"host":"1.1.1.2","ttl":60}' - on the second node

		//Rebuild the key and value to write to etcd
		key1 = "/" + rootKey + "/" + strings.Join(dnsArry, "/") + "/x" + uniqueNumber
		val1 = fmt.Sprintf(`{"host":"%s","ttl":%s}`, hostip, ttl)

		//Initally connect to etcd server and get the interupt channel
		errLease = lease.StartLease(ctx, key1, val1)
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
			case <-ctx.Done(): //Parent ask to quit
				break loop

			}
		}
	retry:
		//TODO: implement backoff retry
	}

}

func reverseArray(a []string) []string {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

// LeaseInf - Enable the controller to start leasing and wait for the signal to change flow
type LeaseInf interface {
	StartLease(ctx context.Context, kvKey string, kvVal string) error
	GetRenewalInteruptChan() (renewalInterupted chan struct{})
	RevokeLease(ctx context.Context) error
}

// InformerInf - Enable the controller to determine what unique key/value to be writen to the Lease
type InformerInf interface {
	GetDNSKeyVal(ctx context.Context) (hostip string, ttl string, uniqueNumber string, err error)
	GetInformerInterupt() (informerInterupted chan struct{})
}
