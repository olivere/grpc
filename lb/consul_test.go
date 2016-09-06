// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package lb

import (
	"io/ioutil"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"

	"google.golang.org/grpc/naming"
)

func TestConsulResolver(t *testing.T) {
	srv := testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	defer srv.Stop()

	cfg := &api.Config{
		Address: srv.HTTPAddr,
	}
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// srv.AddService("service", structs.HealthPassing, []string{"production"})
	err = client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      "service-1",
		Name:    "service",
		Tags:    []string{"production"},
		Address: "192.168.1.100",
		Port:    16384,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      "service-2",
		Name:    "service",
		Tags:    []string{"canary"},
		Address: "192.168.1.101",
		Port:    16385,
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := NewConsulResolver(client, "service", "")
	if err != nil {
		t.Fatal(err)
	}
	w, err := r.Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	updates, err := w.Next()
	if err != nil {
		t.Fatal(err)
	}
	if want, have := 2, len(updates); want != have {
		t.Fatalf("retrieve updates via Next(): want %d, have %d", want, have)
	}
	if updates[0].Addr != "192.168.1.100:16384" && updates[0].Addr != "192.168.1.101:16385" {
		t.Fatalf("1st update Addr: have %q", updates[0].Addr)
	}
	if want, have := naming.Add, updates[0].Op; want != have {
		t.Fatalf("1st update Op: want %v, have %v", want, have)
	}
	if updates[1].Addr != "192.168.1.100:16384" && updates[1].Addr != "192.168.1.101:16385" {
		t.Fatalf("2nd update Addr: have %q", updates[1].Addr)
	}
	if want, have := naming.Add, updates[1].Op; want != have {
		t.Fatalf("2nd update Op: want %v, have %v", want, have)
	}
}
