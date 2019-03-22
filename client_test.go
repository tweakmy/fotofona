package main

import (
	"context"
	"fmt"
	"testing"
)

func TestSum(t *testing.T) {

	if !FuncA() {
		t.Error("Wrong")
	}

}

func TestGrant(t *testing.T) {

	ctx := context.TODO()

	kv, leaseID, lease, _ := NewGrant(ctx, "http://127.0.0.1:2378", 2, "thekey", "thevalue")

	if leaseID < 1 {
		t.Error("Wrong ID")
	}

	ttlresp := lease.TimeToLive()

	err := RenewLease(ctx, lease, leaseID)

	if err != nil {
		t.Error("It should be possible to renew the lease")
	}

	resp, _ := kv.Get(ctx, "thekey")
	val := string(resp.Kvs[0].Value)

	fmt.Println(resp.Kvs[0])
	if val != "thevalue" {
		t.Error("Expecting 'thekey' but got " + val)
	}

	_, _, _, err = NewGrant(ctx, "http://127.0.0.1:3000", 2, "thekey", "thevalue")

	if err == nil {
		t.Error("Should error")
	}

	err2 := RenewLease(ctx, lease, leaseID)

	if err2 == nil {
		t.Error("It should error out")
	}

}
