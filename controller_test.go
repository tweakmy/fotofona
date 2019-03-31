package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

var infTest = InfTest{
	fakehostip:       "1.1.1.1",
	fakettl:          "60",
	fakeUniqueNumber: "1",
	fakeChan:         make(chan struct{}),
}

var alwaysReturnOk = func() bool {
	return true
}

//Test 1: Verify we can received the same key as expected
func TestController(t *testing.T) {

	dnsName := "kubemaster.local"

	//Test 1: Verify we can received the same key as expected
	verify := func(kvkey string, kvval string) bool {

		if kvkey == "/rootkey/local/kubemaster/x1" {
			return true
		}

		t.Error("wrong key: " + kvkey)
		return false
	}

	//Temporary make it work
	infTest.getDNSTestFunc = alwaysReturnOk

	var leaseTest = LeaseTest{
		fakeChan:           make(chan struct{}),
		startLeaseFunc:     verify,
		leaseRevokeRunFunc: alwaysReturnOk,
	}

	ctx, cancel := context.WithCancel(context.TODO())

	go RunController(ctx, "rootkey", dnsName, &leaseTest, &infTest)

	tchan := time.After(2 * time.Second)
	<-tchan
	cancel()

}

//Test 2: Verify that is lease is stopped, it will continue to read the informer
func TestController2(t *testing.T) {

	verifyTest := false

	dnsName := "kubemaster.local"

	doNothingVerify := func(kvkey string, kvval string) bool {
		//Do nothing
		return true
	}

	infTest.getDNSTestFunc = func() bool {
		verifyTest = true
		return true
	}

	var leaseTest = LeaseTest{
		fakeChan:           make(chan struct{}),
		startLeaseFunc:     doNothingVerify,
		leaseRevokeRunFunc: alwaysReturnOk,
	}

	ctx, cancel := context.WithCancel(context.TODO())

	go RunController(ctx, "rootkey", dnsName, &leaseTest, &infTest)

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

//Test 3: Verify that is node informer has changed, it needed a new key for the lease
func TestController3(t *testing.T) {

	verifyTest := false

	dnsName := "kubemaster.local"

	doNothingVerify := func(kvkey string, kvval string) bool {
		//Do nothing
		return true
	}

	infTest.getDNSTestFunc = alwaysReturnOk

	var leaseTest = LeaseTest{
		fakeChan:       make(chan struct{}),
		startLeaseFunc: doNothingVerify,
	}

	leaseTest.leaseRevokeRunFunc = func() bool {
		verifyTest = true
		return true
	}

	ctx, cancel := context.WithCancel(context.TODO())

	go RunController(ctx, "rootkey", dnsName, &leaseTest, &infTest)

	//Trigger that leasing is somehow stopped
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
	fakehostip       string
	fakettl          string
	fakeUniqueNumber string
	err              error
	fakeChan         chan struct{}
	getDNSTestFunc   func() bool
}

// GetDnsKeyVal - this is only for testing
func (i *InfTest) GetDNSKeyVal(ctx context.Context) (hostip string, ttl string, uniqueNumber string, err error) {

	if !i.getDNSTestFunc() {
		return "", "", "", errors.New("fake error")
	}

	return i.fakehostip, i.fakettl, i.fakeUniqueNumber, i.err
}

// GetDnsKeyVal - this is only for testing
func (i *InfTest) GetInformerInterupt() (informerInterupted chan struct{}) {
	return i.fakeChan
}

type LeaseTest struct {
	err                error
	fakeChan           chan struct{}
	startLeaseFunc     func(kvKey string, kvVal string) bool
	leaseRevokeRunFunc func() bool
}

func (l *LeaseTest) StartLease(ctx context.Context, kvKey string, kvVal string) error {

	if l.startLeaseFunc(kvKey, kvVal) {
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
