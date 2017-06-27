// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	lb "github.com/olivere/grpc/lb/consul"
	pb "github.com/olivere/grpc/lb/consul/example/proto/echo"
)

func main() {
	var (
		n = flag.Int("n", 0, "Number of calls to service")
		t = flag.Duration("t", 1*time.Second, "Sleep interval between calls")
	)
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Lookup service in Consul
	cli, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	// Resolver for the "echo" service
	r, err := lb.NewResolver(cli, "echo", "")
	if err != nil {
		log.Fatal(err)
	}

	// Dial options
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	// Enabling WithBlock tells the client to not give up trying to find a server
	opts = append(opts, grpc.WithBlock())
	// However, we're still setting a timeout so that if the server takes too long, we still give up
	opts = append(opts, grpc.WithTimeout(10*time.Second))
	// Add resolver with RoundRobin balancer here
	opts = append(opts, grpc.WithBalancer(grpc.RoundRobin(r)))

	// Notice the blank address
	conn, err := grpc.Dial("", opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewEchoClient(conn)

	timeout := time.Duration(int64(3+(*t).Seconds())) * time.Second
	for i := 0; i < *n; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		req := &pb.EchoRequest{Message: time.Now().Format(time.RFC3339)}
		res, err := client.Echo(ctx, req)
		if err != nil {
			cancel()
			log.Fatal(err)
		}
		fmt.Printf("%v\n", res.Message)
		time.Sleep(*t)
		cancel()
	}
}
