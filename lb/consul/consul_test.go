package consul

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"

	"google.golang.org/grpc/naming"
)

func TestResolver(t *testing.T) {
	srv, err := testutil.NewTestServerConfigT(t, func(c *testutil.TestServerConfig) {
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	cfg := &api.Config{
		Address: srv.HTTPAddr,
	}
	client, err := api.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	err = client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      "service-1",
		Name:    "service",
		Address: "192.168.1.100",
		Port:    16384,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      "service-2",
		Name:    "service",
		Address: "192.168.1.101",
		Port:    16385,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      "service-3",
		Name:    "another-service",
		Address: "192.168.1.102",
		Port:    16386,
	})
	if err != nil {
		t.Fatal(err)
	}

	sd := NewStreamDemux(client, time.Minute*5)
	if err != nil {
		t.Fatal(err)
	}
	r, err := sd.NewResolver("service", "")
	if err != nil {
		t.Fatalf("failed to create resolver")
	}
	ra, err := sd.NewResolver("another-service", "")
	if err != nil {
		t.Fatalf("failed to create resolver")
	}

	w, err := r.Resolve("service")
	if err != nil {
		t.Fatal(err)
	}
	wa, err := ra.Resolve("another-service")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	defer wa.Close()

	updates, err := w.Next()
	if err != nil {
		t.Fatal(err)
	}
	updatesa, err := wa.Next()
	if err != nil {
		t.Fatal(err)
	}

	if want, have := 2, len(updates); want != have {
		t.Fatalf("retrieve updates via Next(): want %d, have %d", want, have)
	}
	if want, have := 1, len(updatesa); want != have {
		t.Fatalf("retrieve updates via Next(): want %d, have %d", want, have)
	}
	if updates[0].Addr != "192.168.1.100:16384" && updates[0].Addr != "192.168.1.101:16385" {
		t.Fatalf("1st update Addr for service: have %q", updates[0].Addr)
	}
	if want, have := naming.Add, updates[0].Op; want != have {
		t.Fatalf("1st update Op for service: want %v, have %v", want, have)
	}
	if updates[1].Addr != "192.168.1.100:16384" && updates[1].Addr != "192.168.1.101:16385" {
		t.Fatalf("2nd update Addr for service: have %q", updates[1].Addr)
	}
	if want, have := naming.Add, updates[1].Op; want != have {
		t.Fatalf("2nd update Op for service: want %v, have %v", want, have)
	}
	if updatesa[0].Addr != "192.168.1.102:16386" {
		t.Fatalf("1st update Addr for another-service: have %q", updatesa[0].Addr)
	}
	if want, have := naming.Add, updatesa[0].Op; want != have {
		t.Fatalf("1st update Op for another-service: want %v, have %v", want, have)
	}
}
