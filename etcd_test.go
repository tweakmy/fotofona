package main

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
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

	//Delay for sometime for the etcd server to start
	time.Sleep(2 * time.Second)

	return cmd

}

//Verify that ETCD lease is working
func TestEtcdLease(t *testing.T) {

	tc := struct {
		inputCond struct {
			leaseTime int
			entries   []Entry
		}
		expected []string
	}{
		inputCond: struct {
			leaseTime int
			entries   []Entry
		}{
			entries: []Entry{
				Entry{Key: "/key1", Val: "Val1"},
				Entry{Key: "/key2", Val: "Val2"},
			},
			leaseTime: 5,
		},
		expected: []string{
			"/key1" + ":" + "Val1",
			"/key2" + ":" + "Val2",
		},
	}

	cmd := SetupEtcdServer()
	defer cmd.Process.Kill()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://localhost:2378"},
		DialTimeout: 2 * time.Second,
	})

	if err != nil {
		t.Error(err.Error())
		return
	}

	etcd := NewEtcdLease(cli)

	entries := tc.inputCond.entries

	etcd.InitLease(ctx, entries, tc.inputCond.leaseTime)

	resp, err := cli.Get(ctx, "/key1", clientv3.WithRange("/key2"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	if len(resp.Kvs) < 1 {
		t.Error("No result was found")
	}

	for i, kv := range resp.Kvs {
		outcome := fmt.Sprintf("%s:", kv.Key) + fmt.Sprintf("%s", kv.Value)
		if tc.expected[i] != outcome {
			t.Errorf("When read key, expected: %d %s but outcome: %s", i, tc.expected[i], outcome)
		}
	}

	time.Sleep(time.Duration(tc.inputCond.leaseTime+1) * time.Second)

	resp2, err := cli.Get(ctx, "/key1", clientv3.WithRange("/key2"))
	if err != nil {
		t.Error(err.Error())
		return
	}
	expected2 := fmt.Sprint(resp2.Kvs)
	if expected2 != "[]" {
		t.Errorf("Expected the key to automatically removed, key after the expiry %s", expected2)
	}

}

func TestEtcdRenewLease(t *testing.T) {

	tc := struct {
		inputCond struct {
			leaseTime int
			entries   []Entry
		}
		expected []string
	}{
		inputCond: struct {
			leaseTime int
			entries   []Entry
		}{
			entries: []Entry{
				Entry{Key: "/key1", Val: "Val2"},
				Entry{Key: "/key2", Val: "Val1"},
			},
			leaseTime: 5,
		},
		expected: []string{
			"/key1" + ":" + "Val2",
			"/key2" + ":" + "Val1",
		},
	}

	cmd := SetupEtcdServer()
	defer cmd.Process.Kill()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://localhost:2378"},
		DialTimeout: 2 * time.Second,
	})

	if err != nil {
		t.Error(err.Error())
		return
	}

	etcd := NewEtcdLease(cli)

	entries := tc.inputCond.entries

	if etcd.InitLease(ctx, entries, tc.inputCond.leaseTime) != nil {
		t.Error(err.Error())
		return
	}

	interupt, err := etcd.RenewLease(ctx)

	if err != nil {
		t.Error(err.Error())
		return
	}

	fmt.Println("Verify key is still being kept after the expiry time the lease is renewed")
	time.Sleep(time.Duration(tc.inputCond.leaseTime+1) * time.Second)

	resp, err := cli.Get(ctx, "/key1", clientv3.WithRange("/key2"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	//verify key value
	for i, kv := range resp.Kvs {
		outcome := fmt.Sprintf("%s:", kv.Key) + fmt.Sprintf("%s", kv.Value)
		if tc.expected[i] != outcome {
			t.Errorf("When read key, expected: %d %s but outcome: %s", i, tc.expected[i], outcome)
		}
	}

	fmt.Println("Verify that after the etcd client is failing, the interupt is send")
	cmd.Process.Kill()

	select {
	case <-interupt:
		return
	case <-time.After(5 * time.Second):
		t.Error("Did not received the interupt")
	}

}

func TestRevokeLease(t *testing.T) {

	tc := struct {
		inputCond struct {
			leaseTime int
			entries   []Entry
		}
		expected []string
	}{
		inputCond: struct {
			leaseTime int
			entries   []Entry
		}{
			entries: []Entry{
				Entry{Key: "/key1", Val: "Val2"},
				Entry{Key: "/key2", Val: "Val1"},
			},
			leaseTime: 5,
		},
		expected: []string{
			"/key1" + ":" + "Val2",
			"/key2" + ":" + "Val1",
		},
	}

	cmd := SetupEtcdServer()
	defer cmd.Process.Kill()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://localhost:2378"},
		DialTimeout: 2 * time.Second,
	})

	if err != nil {
		t.Error(err.Error())
		return
	}

	etcd := NewEtcdLease(cli)

	entries := tc.inputCond.entries

	if etcd.InitLease(ctx, entries, tc.inputCond.leaseTime) != nil {
		t.Error(err.Error())
		return
	}

	interupt, err := etcd.RenewLease(ctx)

	time.Sleep(2 * time.Second)

	if etcd.RevokeLease(ctx) != nil {
		t.Error(err.Error())
		return
	}

	select {
	case <-interupt:
		return
	case <-time.After(5 * time.Second):
		t.Error("Did not received the interupt")
	}

}
