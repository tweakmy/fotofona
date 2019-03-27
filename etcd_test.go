package main

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"
)

// func TestSum(t *testing.T) {

// 	if !FuncA() {
// 		t.Error("Wrong")
// 	}

// }

func SetupEtcdServer() *exec.Cmd {

	cmd := exec.Command("etcd", "--listen-client-urls=http://localhost:2378",
		"--advertise-client-urls=http://localhost:2378", "--listen-peer-urls=http://localhost:2382")

	cmd.Dir = "./test-etcd" //Setup the data directory in ./test-etcd

	error := cmd.Start()
	if error != nil {
		fmt.Println(error)
	}

	return cmd

}

func TestGrant(t *testing.T) {

	cmd := SetupEtcdServer()

	ctx := context.TODO()

	kv, leaseID, lease, _ := NewGrant(ctx, "http://127.0.0.1:2378", 2, "thekey", "thevalue")

	if leaseID < 1 {
		t.Error("Wrong ID")
	}

	//fmt.Println(leaseID)
	//ttlresp := lease.TimeToLive(ctx, leaseID)

	err := RenewLeaseOnce(ctx, lease, leaseID)

	if err != nil {
		t.Error("It should be possible to renew the lease")
	}

	resp, _ := kv.Get(ctx, "thekey")
	val := string(resp.Kvs[0].Value)

	//fmt.Println(resp.Kvs[0])
	if val != "thevalue" {
		t.Error("Expecting 'thekey' but got " + val)
	}

	_, _, _, err = NewGrant(ctx, "http://127.0.0.1:3000", 2, "thekey", "thevalue")

	if err == nil {
		t.Error("It should error as it cannot connect to port 3000")
	}

	err2 := RenewLeaseOnce(ctx, lease, leaseID)

	if err2 == nil {
		t.Error("It should error out")
	}

	err5 := cmd.Process.Kill()

	if err5 != nil {
		t.Error("Cannot kill the etcd server:" + err5.Error())
	}

	renewInteruptChan, _ := RenewLease(ctx, lease, leaseID, 2)

	select {
	case <-renewInteruptChan:
		break
	case <-time.After(5 * time.Second):
		t.Error("It supposed to send interupt")
	}

}
