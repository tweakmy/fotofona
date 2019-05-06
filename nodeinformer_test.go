package main

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
)

// NodeAddress struct {
// 	// Node address type, one of Hostname, ExternalIP or InternalIP.
// 	Type NodeAddressType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=NodeAddressType"`
// 	// The node address.
// 	Address

type clientops struct {
	//Possible node operation on the kube cluster: ADD, UPDATE, DELETE, NONE
	addOrdelOrUpdate string

	//What are the node that is being updated
	whatNode *v1.Node

	//time allow to wait for next operation after this operation
	timewaitinSec time.Time
}

//Specify what are the format of the TDD
type tddInformerCond struct {

	//What was the initial node before informer was started
	initialNodes []*v1.Node

	//clientops
	clientops []clientops

	//Specific the expected output
	want [][]string
}

func TestInformer(t *testing.T) {

	node1 := NewMasterNode("node1", "10.0.0.1", "Ready")
	node2 := NewMasterNode("node2", "10.0.0.3", "Ready")
	var TestCondition = tddInformerCond{
		initialNodes: []*v1.Node{
			node1,
			node2},

		clientops: []clientops{
			clientops{addOrdelOrUpdate: "NONE", whatNode: nil},
			clientops{addOrdelOrUpdate: "ADD", whatNode: NewMasterNode("node3", "10.0.0.2", "Ready")},
			clientops{addOrdelOrUpdate: "DELETE", whatNode: node1},
		},

		want: [][]string{
			[]string{"10.0.0.1", "10.0.0.3"},
			[]string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
			[]string{"10.0.0.2", "10.0.0.3"},
		},
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	nodes := []runtime.Object{}

	//Cast it to the runtime.Object
	for _, v := range TestCondition.initialNodes {
		nodes = append(nodes, v)
	}

	// Create the fake client.
	fakeClient := fake.NewSimpleClientset(nodes...)
	inf := Informer{
		clientset:         fakeClient,
		updateHostIPsChan: make(chan struct{}),
		queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
	sigChan := inf.GetInformerInterupt()

	//Start the informer to read data
	go inf.Start(ctx)

forloop:
	for i, op := range TestCondition.clientops {
		time.Sleep(time.Second)
		//time.Sleep(time.Second) //sleep 1 second
		fmt.Printf("Evaluating test informer case %d\n", i)

		switch op.addOrdelOrUpdate {
		case "NONE":
			outcomeArry := inf.GetHostIPs()
			sort.Strings(outcomeArry)
			outcome := fmt.Sprint(outcomeArry)

			expected := fmt.Sprint(TestCondition.want[i])

			if outcome != expected {
				t.Errorf("test item %d hostips outcome %s vs expected %s", i, outcome, expected)
				break forloop
			}

		case "ADD":

			fakeClient.CoreV1().Nodes().Create(op.whatNode)
			<-sigChan //Then wait for the changes to be notified
			outcomeArry := inf.GetHostIPs()
			sort.Strings(outcomeArry)
			outcome := fmt.Sprint(outcomeArry)

			expected := fmt.Sprint(TestCondition.want[i])
			if outcome != expected {
				t.Errorf("test item %d hostips outcome %s vs expected %s", i, outcome, expected)
				break forloop
			}
		case "DELETE":

			fakeClient.CoreV1().Nodes().Delete(node1.Name, nil)
			<-sigChan //Then wait for the changes to be notified
			outcomeArry := inf.GetHostIPs()
			sort.Strings(outcomeArry)
			outcome := fmt.Sprint(outcomeArry)

			expectedArry := TestCondition.want[i]
			sort.Strings(outcomeArry)
			expected := fmt.Sprint(expectedArry)
			if outcome != expected {
				t.Errorf("test item %d hostips outcome %s vs expected %s", i, outcome, expected)
				break forloop
			}
		}
	}

}

func NewMasterNode(nodeName string, ipaddress string, statusphase string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
		},
		Status: v1.NodeStatus{
			Phase: v1.NodePhase(statusphase),
			Addresses: []v1.NodeAddress{
				v1.NodeAddress{
					Type:    v1.NodeAddressType("InternalIP"),
					Address: ipaddress,
				},
			},
		},
	}
}
