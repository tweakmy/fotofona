package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

var infTest = InfTest{
	fakehostip: []string{"1.1.1.1"},
	fakeChan:   make(chan struct{}),
	getDNSTestFunc: func() bool {
		return true //Defaults to return ok everytime
	},
}

var leaseTest = LeaseTest{
	fakeChan: make(chan struct{}),
	startLeaseFunc: func(entries []Entry) bool {
		return true //Defaults to return ok everytime
	},
	leaseRevokeRunFunc: func() bool {
		return true //Defaults to return ok everytime
	},
}

//Test 1: Verify we can received the same key as expected
func TestController1(t *testing.T) {

	dnsName := "kubemaster.local"

	//This is the only way to verify whether we are getting the right key and value
	leaseTest.startLeaseFunc = func(entries []Entry) bool {

		if entries[0].Key == "/rootkey/local/kubemaster/x1" &&
			entries[0].Val == `{"host":"1.1.1.1","ttl":10}` {
			return true
		}

		t.Error("wrong key: " + fmt.Sprint(entries))
		return false
	}

	ctx, cancel := context.WithCancel(context.TODO())

	go RunController(ctx, "rootkey", dnsName, 10, &leaseTest, &infTest)

	tchan := time.After(2 * time.Second)
	<-tchan
	cancel()

}

//Test 2: Verify that is lease is stopped, it will continue to read the informer
func TestController2(t *testing.T) {

	verifyTest := false

	dnsName := "kubemaster.local"

	ctx, cancel := context.WithCancel(context.TODO())

	infTest.getDNSTestFunc = func() bool {
		verifyTest = true
		return true
	}

	go RunController(ctx, "rootkey", dnsName, 10, &leaseTest, &infTest)

	//Trigger that leasing is somehow stopped
	leaseTest.fakeChan <- struct{}{}

	select {
	case <-time.After(1 * time.Second):
		if !verifyTest {
			t.Error("Should terminate and get to read DNS record")
		}
	}

	tchan := time.After(2 * time.Second)
	<-tchan
	cancel()

}

//Test 3: Verify that is node informer has changed, it will revoke the lease
func TestController3(t *testing.T) {

	verifyTest := false

	dnsName := "kubemaster.local"

	ctx, cancel := context.WithCancel(context.TODO())

	leaseTest.leaseRevokeRunFunc = func() bool {
		verifyTest = true

		cancel()
		return true
	}

	go RunController(ctx, "rootkey", dnsName, 10, &leaseTest, &infTest)

	//Wait for a second
	time.Sleep(1 * time.Second)

	//Trigger that informer has changed
	infTest.fakeChan <- struct{}{}

	select {
	case <-time.After(1 * time.Second):
		if !verifyTest {
			t.Error("It should perform a lease revoke")
		}
	}

	tchan := time.After(2 * time.Second)
	<-tchan
	cancel()

}

type InfTest struct {
	fakehostip     []string
	err            error
	fakeChan       chan struct{}
	getDNSTestFunc func() bool
}

// GetDnsKeyVal - this is only for testing
func (i *InfTest) GetHostIPs(ctx context.Context) (hostip []string, err error) {

	if !i.getDNSTestFunc() {
		return nil, i.err
	}

	return i.fakehostip, nil
}

// GetDnsKeyVal - this is only for testing
func (i *InfTest) GetInformerInterupt() (informerInterupted chan struct{}) {
	return i.fakeChan
}

type LeaseTest struct {
	err                error
	fakeChan           chan struct{}
	startLeaseFunc     func(entries []Entry) bool
	leaseRevokeRunFunc func() bool
}

func (l *LeaseTest) StartLease(ctx context.Context, entries []Entry) error {

	if l.startLeaseFunc(entries) {
		return nil
	}

	return l.err

}

func (l *LeaseTest) GetRenewalInteruptChan() (renewalInterupted chan struct{}) {
	return l.fakeChan
}

func (l *LeaseTest) RevokeLease(ctx context.Context) error {
	if !l.leaseRevokeRunFunc() {
		return l.err
	}

	return nil
}
