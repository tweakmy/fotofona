package main

import (
	"context"
	"testing"
	"time"
)

type testCondCtrl struct {
	verifyGate bool
	rootKey    string
	DNSname    string
	DNSTTL     int
	informer   *infTest
	leaser     *leaseTest
}

func runControllerFunc(tc testCondCtrl) context.CancelFunc {

	ctx, cancel := context.WithCancel(context.Background())

	go RunController(ctx, tc.rootKey, tc.DNSname, tc.DNSTTL, tc.leaser, tc.informer)

	return cancel
}

// Verify the Entries are correctly converted and the Leaser will get the correct value
func TestControllerVerifyEntriesOk(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	tc := testCondCtrl{
		rootKey: "rootkey",
		DNSname: "kubemaster.local",
		DNSTTL:  60,
		informer: &infTest{
			fakehostip: []string{"1.1.1.1"}, //Assumed there is only 1 Node ip available
			fakeChan:   make(chan struct{}),
			getDNSTestFunc: func() bool {
				return true //Defaults to return ok everytime
			},
		},
		leaser: &leaseTest{
			fakeChan: make(chan struct{}),
			startLeaseFunc: func(entries []Entry, leaseTimeInSec int) bool {
				expectedKey := "/rootkey/local/kubemaster/x1"
				expectedVal := `{"host":"1.1.1.1","ttl":60}`

				if entries[0].Key != expectedKey || entries[0].Val != expectedVal {
					t.Errorf("Expected key: %s and value %s but go key %s and value %s",
						expectedKey, expectedVal, entries[0].Key, entries[0].Val)
				}

				if leaseTimeInSec != 30 {
					t.Errorf("Expected lease time expected %d vs outcome %d", 30, leaseTimeInSec)
				}

				cancel() //Stop further processing the controller
				return true

			},
			leaseRevokeRunFunc: func() bool {
				return true //Defaults to return ok everytime
			},
		},
	}

	go RunController(ctx, tc.rootKey, tc.DNSname, tc.DNSTTL, tc.leaser, tc.informer)

	time.Sleep(2 * time.Second)
	cancel()
}

// Verify that if lease is stopped, it will read the informer ip address again
func TestControllerLeaseStopOK(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	//Verify the translated key is correct
	tc := testCondCtrl{
		rootKey: "rootkey",
		DNSname: "kubemaster.local",
		DNSTTL:  10,
		informer: &infTest{
			fakehostip: []string{"2.2.1.11"},
			fakeChan:   make(chan struct{}),
			getDNSTestFunc: func() bool {
				return true
			},
		},
		leaser: &leaseTest{
			fakeChan: make(chan struct{}),
			startLeaseFunc: func(entries []Entry, leaseTimeInSec int) bool {
				expectedKey := "/rootkey/local/kubemaster/x1"
				expectedVal := `{"host":"2.2.1.11","ttl":10}`

				if entries[0].Key != expectedKey || entries[0].Val != expectedVal {
					t.Errorf("Expected key: %s and value %s but go key %s and value %s",
						expectedKey, expectedVal, entries[0].Key, entries[0].Val)
				}

				return true

			},
			leaseRevokeRunFunc: func() bool {
				return true //Defaults to return ok everytime
			},
		},
	}

	//If the func runs a second time, it should then check the verify gate
	tc.informer.getDNSTestFunc = func() bool {

		if tc.informer.readCount > 1 {

			tc.verifyGate = true
			cancel()
		}
		return true
	}

	go RunController(ctx, tc.rootKey, tc.DNSname, tc.DNSTTL, tc.leaser, tc.informer)

	tchan := time.After(2 * time.Second)
	<-tchan

	tc.leaser.fakeChan <- struct{}{}

	select {
	case <-time.After(5 * time.Second):
		if !tc.verifyGate {
			t.Error("It is not reading the hostsips")
		}
	}

	cancel()
}

// Verify informer has changed, it will revoke the lease
func TestControllerinformerChangeOk(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())

	//Verify the translated key is correct
	tc := testCondCtrl{
		rootKey: "rootkey",
		DNSname: "kubemaster.local",
		DNSTTL:  10,
		informer: &infTest{
			fakehostip: []string{"2.2.1.11"},
			fakeChan:   make(chan struct{}),
			getDNSTestFunc: func() bool {
				return true
			},
		},
		leaser: &leaseTest{
			fakeChan: make(chan struct{}),
			startLeaseFunc: func(entries []Entry, leaseTimeInSec int) bool {
				expectedKey := "/rootkey/local/kubemaster/x1"
				expectedVal := `{"host":"2.2.1.11","ttl":10}`

				if entries[0].Key != expectedKey || entries[0].Val != expectedVal {
					t.Errorf("Expected key: %s and value %s but go key %s and value %s",
						expectedKey, expectedVal, entries[0].Key, entries[0].Val)
				}

				return true

			},
			leaseRevokeRunFunc: func() bool {
				return true //Defaults to return ok everytime
			},
		},
	}

	//If the func runs a second time, it should then check the verify gate
	tc.leaser.leaseRevokeRunFunc = func() bool {

		tc.verifyGate = true
		cancel()

		return true
	}

	go RunController(ctx, tc.rootKey, tc.DNSname, tc.DNSTTL, tc.leaser, tc.informer)

	tchan := time.After(2 * time.Second)
	<-tchan

	tc.informer.fakeChan <- struct{}{}

	select {
	case <-time.After(5 * time.Second):
		if !tc.verifyGate {
			t.Error("It should revoke the lease")
		}
	}

	cancel()
}

type infTest struct {
	fakehostip     []string
	err            error
	fakeChan       chan struct{}
	getDNSTestFunc func() bool
	readCount      int
}

func (i *infTest) Start(ctx context.Context) {
	//Not sure yet, what this should be doing
}

// GetDnsKeyVal - this is only for testing
func (i *infTest) GetHostIPs(ctx context.Context) (hostip []string, err error) {

	i.readCount++

	if !i.getDNSTestFunc() {
		return nil, i.err
	}

	return i.fakehostip, nil
}

// GetDnsKeyVal - this is only for testing
func (i *infTest) GetInformerInterupt() (informerInterupted chan struct{}) {
	return i.fakeChan
}

type leaseTest struct {
	err                error
	fakeChan           chan struct{}
	startLeaseFunc     func(entries []Entry, leaseTimeInSec int) bool
	leaseRevokeRunFunc func() bool
}

func (l *leaseTest) StartLease(ctx context.Context, entries []Entry, leaseTimeInSec int) error {

	if l.startLeaseFunc(entries, leaseTimeInSec) {
		return nil
	}

	return l.err

}

func (l *leaseTest) GetRenewalInteruptChan() (renewalInterupted chan struct{}) {
	return l.fakeChan
}

func (l *leaseTest) RevokeLease(ctx context.Context) error {
	if !l.leaseRevokeRunFunc() {
		return l.err
	}

	return nil
}
