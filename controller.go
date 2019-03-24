package main

// RunController - Run the loop to periodically write the loop
func RunController(refreshrate int) {
	//Todo: Read kubernetes master for the master nodes

	// etcdctl put /skydns/local/skydns/x1 '{"host":"1.1.1.1","ttl":60}'
	// etcdctl put /skydns/local/skydns/x2 '{"host":"1.1.1.2","ttl":60}'

	for {
		//Initally connect to etcd server and get the interupt channel

		select {}
	}

}
