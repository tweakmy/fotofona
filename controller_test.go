package main

import (
	"context"
	"testing"
	"time"
)

type InfTest struct {
	fakehostip       string
	fakettl          string
	fakeUniqueNumber string
	err              error
	fakeChan         chan struct{}
}

func TestController(t *testing.T) {

	dnsName := "kubemaster.local"

	infTest := InfTest{
		fakehostip:       "1.1.1.1",
		fakettl:          "60",
		fakeUniqueNumber: "1",
		fakeChan:         make(chan struct{}),
	}

	verify := func(kvkey string, kvval string) bool {

		if kvkey == "/rootkey/local/kubemaster/x1" {
			return true
		}

		t.Error("wrong key: " + kvkey)
		return false
	}

	leaseTest := LeaseTest{
		fakeChan: make(chan struct{}),
		testFunc: verify,
	}

	ctx, cancel := context.WithCancel(context.TODO())

	go RunController(ctx, "rootkey", dnsName, &leaseTest, &infTest)

	tchan := time.After(2 * time.Second)
	<-tchan
	cancel()
}

// GetDnsKeyVal - this is only for testing
func (i *InfTest) GetDNSKeyVal(ctx context.Context) (hostip string, ttl string, uniqueNumber string, err error) {
	return i.fakehostip, i.fakettl, i.fakeUniqueNumber, i.err
}

// GetDnsKeyVal - this is only for testing
func (i *InfTest) GetInformerInterupt() (informerInterupted chan struct{}) {
	return i.fakeChan
}

type LeaseTest struct {
	err      error
	fakeChan chan struct{}
	testFunc func(kvKey string, kvVal string) bool
}

func (l *LeaseTest) StartLease(ctx context.Context, kvKey string, kvVal string) error {

	if l.testFunc(kvKey, kvVal) {
		return nil
	}

	return l.err

}

func (l *LeaseTest) GetRenewalInteruptChan() (renewalInterupted chan struct{}) {
	return l.fakeChan
}

func (l *LeaseTest) RevokeLease(ctx context.Context) error {
	return l.err
}
