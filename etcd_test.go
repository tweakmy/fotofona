package main

import (
	"context"
	"flag"
	"fmt"
	"os"
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

	etcdStartDir := "./test-etcd"

	cmd := exec.Command("etcd", "--listen-client-urls=http://localhost:2378",
		"--advertise-client-urls=http://localhost:2378", "--listen-peer-urls=http://localhost:2382")

	cmd.Dir = etcdStartDir //Setup the data directory in ./test-etcd

	//remove the data directory to avoid confusion
	error := os.RemoveAll(etcdStartDir + "/default.etcd")
	if error != nil {
		fmt.Println(error)
	}

	error = cmd.Start()
	if error != nil {
		fmt.Println(error)
	}

	//Delay for sometime for the etcd server to start
	time.Sleep(5 * time.Second)

	return cmd

}

//Verify that ETCD lease is working
func TestEtcdLease(t *testing.T) {

	flag.Set("alsologtostderr", fmt.Sprintf("%t", true))
	var logLevel string
	flag.StringVar(&logLevel, "logLevel", "2", "test")
	flag.Lookup("v").Value.Set(logLevel)

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

	resp, err := cli.Get(ctx, "/key", clientv3.WithPrefix())
	if err != nil {
		t.Error(err.Error())
		return
	}

	if len(resp.Kvs) != 2 {
		t.Error("There should be 2 keys", resp.Kvs)
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

	resp, err := cli.Get(ctx, "/key", clientv3.WithPrefix())
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

func TestEtcdRevokeLease(t *testing.T) {

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

func startDNS() *exec.Cmd {

	cmd := exec.Command("coredns", "-dns.port=8053", "-conf=Corefile")
	cmd.Dir = ".test-dns"

	error := cmd.Start()
	if error != nil {
		fmt.Println(error)
	}

	return cmd
}

// Verify that the entries is what is expected by coredns every release
func TestCoreDNS(t *testing.T) {

	entries :=
		[]Entry{
			Entry{Key: "/skydns/local/skydns/x1", Val: "{\"host\":\"1.1.1.1\",\"ttl\":60}"},
			Entry{Key: "/skydns/local/skydns/x2", Val: "{\"host\":\"1.1.1.2\",\"ttl\":60}"},
		}

	expectedOutcome := `1.1.1.2
1.1.1.1
`
	expectedOutcome2 := `1.1.1.1
1.1.1.2
`

	//Start the CoreDNS
	cmd := startDNS()
	defer cmd.Process.Kill()

	//Start DNS Server
	cmd2 := SetupEtcdServer()
	defer cmd2.Process.Kill()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://localhost:2378"},
		DialTimeout: 2 * time.Second,
	})

	if err != nil {
		t.Error(err.Error())
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	etcd := NewEtcdLease(cli)
	err2 := etcd.InitLease(ctx, entries, 300)

	if err2 != nil {
		t.Error(err.Error())
		return
	}

	time.Sleep(5 * time.Second)

	out, err := exec.Command("dig", "-p", "8053", "@127.0.0.1", "skydns.local", "+short").Output()

	if err != nil {
		t.Error(err.Error())
		return
	}

	cmdlineOut := string(out)
	if cmdlineOut != expectedOutcome && cmdlineOut != expectedOutcome2 {
		fmt.Println(cmdlineOut)
		t.Error("Wrong DNS ip resolved")
	}

}
