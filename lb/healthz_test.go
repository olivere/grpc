// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package lb

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc/naming"
	// "google.golang.org/grpc/naming"
)

func TestHealthzResolver(t *testing.T) {
	var endpoints []HealthzEndpoint

	srv1status := http.StatusOK
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(srv1status)
	}))
	defer srv1.Close()
	endpoints = append(endpoints, HealthzEndpoint{
		Addr:     "127.0.0.1:10000",
		CheckURL: srv1.URL,
	})

	srv2status := http.StatusOK
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(srv2status)
	}))
	defer srv2.Close()
	endpoints = append(endpoints, HealthzEndpoint{
		Addr:     "127.0.0.1:10001",
		CheckURL: srv2.URL,
	})

	// Decrease update interval to speed up test
	healthzDefaultUpdateInterval = 1 * time.Second

	// Setup Resolver
	r, err := NewHealthzResolver(endpoints...)
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
	if updates[0].Addr != "127.0.0.1:10000" && updates[0].Addr != "127.0.0.1:10001" {
		t.Errorf("1st update Addr: have %q", updates[0].Addr)
	}
	if want, have := naming.Add, updates[0].Op; want != have {
		t.Errorf("1st update Op: want %v, have %v", want, have)
	}
	if updates[1].Addr != "127.0.0.1:10000" && updates[1].Addr != "127.0.0.1:10001" {
		t.Errorf("2nd update Addr: have %q", updates[1].Addr)
	}
	if want, have := naming.Add, updates[1].Op; want != have {
		t.Errorf("2nd update Op: want %v, have %v", want, have)
	}

	// Disable srv1, and we should receive a Delete op
	srv1status = http.StatusBadGateway
	updates, err = w.Next()
	if err != nil {
		t.Fatal(err)
	}
	if want, have := 1, len(updates); want != have {
		t.Fatalf("retrieve updates via Next(): want %d, have %d", want, have)
	}
	if updates[0].Addr != "127.0.0.1:10000" {
		t.Errorf("1st update Addr: have %q", updates[0].Addr)
	}
	if want, have := naming.Delete, updates[0].Op; want != have {
		t.Errorf("1st update Op: want %v, have %v", want, have)
	}

	// Enable srv1 again, and we should receive an Add op
	srv1status = http.StatusOK
	updates, err = w.Next()
	if err != nil {
		t.Fatal(err)
	}
	if want, have := 1, len(updates); want != have {
		t.Fatalf("retrieve updates via Next(): want %d, have %d", want, have)
	}
	if updates[0].Addr != "127.0.0.1:10000" {
		t.Errorf("1st update Addr: have %q", updates[0].Addr)
	}
	if want, have := naming.Add, updates[0].Op; want != have {
		t.Errorf("1st update Op: want %v, have %v", want, have)
	}
}
