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

	return cmd

}

//Verify that ETCD lease is working
func TestEtcdLease(t *testing.T) {

	tc := struct {
		entries  []Entry
		expected []string
	}{
		entries: []Entry{
			Entry{Key: "/key1", Val: "Val1"},
			Entry{Key: "/key2", Val: "Val2"},
		},
		expected: []string{
			"/key1" + ":" + "Val1",
			"/key2" + ":" + "Val2",
		},
	}

	cmd := SetupEtcdServer()

	ctx, cancel := context.WithCancel(context.Background())

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://localhost:2378"},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Error(err.Error())
		cmd.Process.Kill()
		cancel()
		return
	}

	etcd := NewEtcdLease(cli)

	entries := tc.entries

	etcd.StartLease(ctx, entries, 10)
	etcd.RenewLease(ctx)

	resp, err := cli.Get(ctx, "/Key")
	if err != nil {
		t.Error(err.Error())
		cmd.Process.Kill()
		cancel()
		return
	}

	if len(resp.Kvs) < 1 {
		t.Error("No result was found")
	}

	for i, kv := range resp.Kvs {
		outcome := fmt.Sprintf("%s:", kv.Key) + fmt.Sprintf("%s:", kv.Value)
		if tc.expected[i] != outcome {
			t.Errorf("Expected: %d %s but outcome: %s", i, tc.expected[i], outcome)
		}
	}

	//Kill the etcd server
	cmd.Process.Kill()
	cancel()
}
