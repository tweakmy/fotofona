package main

import (
	"context"
	"fmt"
	"strings"
)

// RunController - Run the loop to periodically write the loop
func RunController(ctx context.Context, rootKey string, dnsname string, dnsTTL int, lease LeaseInf, inf InformerInf) {

	//Dns name remain constant over long period of time
	dnsArry := reverseArray(strings.Split(dnsname, "."))

loop:
	for {

		var errLease error
		var entries []Entry

		hostips, err := inf.GetHostIPs(ctx)
		if err != nil {
			goto retry
		}

		// etcdctl put /skydns/local/skydns/x1 '{"host":"1.1.1.1","ttl":60}' - on the first node
		// etcdctl put /skydns/local/skydns/x2 '{"host":"1.1.1.2","ttl":60}' - on the second node

		//Rebuild the key and value to write to etcd
		entries = make([]Entry, len(hostips))

		for i := range entries {
			entries[i].Key = fmt.Sprintf("/%s/%s/x%d", rootKey, strings.Join(dnsArry, "/"), i+1)
			entries[i].Val = fmt.Sprintf(`{"host":"%s","ttl":%d}`, hostips[i], dnsTTL)
		}

		//Initally connect to etcd server and get the interupt channel
		errLease = lease.StartLease(ctx, entries)

		if errLease == nil {
			select {
			case <-lease.GetRenewalInteruptChan():
				fmt.Println("Controller detected a interuption on the renewal")
				//Do nothing

			case <-inf.GetInformerInterupt():

				err := lease.RevokeLease(ctx)
				if err != nil {
					//What is handling here... I am not sure yet
					panic("Not properly implemented as yet")
				}

				fmt.Println("Controller revoking the lease")

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
	StartLease(ctx context.Context, entries []Entry) error
	GetRenewalInteruptChan() (renewalInterupted chan struct{})
	RevokeLease(ctx context.Context) error
}

// InformerInf - Enable the controller to determine what unique key/value to be writen to the Lease
type InformerInf interface {
	GetHostIPs(ctx context.Context) (hostip []string, err error)
	GetInformerInterupt() (informerInterupted chan struct{})
}
