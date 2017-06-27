// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/hashicorp/consul/api"
	"github.com/olivere/randport"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/olivere/grpc/lb/consul/example/proto/echo"
)

func main() {
	var (
		addr = flag.String("addr", "", "gRPC address to bind to (default: 127.0.0.1:<random>)")
	)
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(*addr) == 0 {
		*addr = fmt.Sprintf("127.0.0.1:%d", randport.Get())
	}
	address, portstr, err := net.SplitHostPort(*addr)
	if err != nil {
		log.Fatal(err)
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Binding to %s", *addr)

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	// Register service with Consul
	cli, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}
	reg := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("echo-%s", uuid.NewV4().String()),
		Name:    "echo",
		Tags:    []string{},
		Address: address,
		Port:    port,
	}
	err = cli.Agent().ServiceRegister(reg)
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Agent().ServiceDeregister(reg.ID)

	errc := make(chan error, 1)

	// Create and serve gRPC server
	go func() {
		var opts []grpc.ServerOption
		// Add e.g. TLS config or credentials here
		// opts = append(opts, grpc.Creds(...))

		srv := grpc.NewServer(opts...)
		pb.RegisterEchoServer(srv, newServer())
		errc <- srv.Serve(lis)
	}()

	// Wait for signal
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("signal %v", <-c)
		errc <- nil
	}()

	if err := <-errc; err != nil {
		log.Fatal(err)
	} else {
		log.Println("Done")
	}
}

// -- Server implementation --

type echoServer struct{}

func newServer() *echoServer {
	return &echoServer{}
}

func (s *echoServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	log.Printf("Received %q", req.Message)
	res := &pb.EchoResponse{Message: req.Message}
	return res, nil
}
