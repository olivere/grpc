// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package lb

import (
	"testing"
	"time"

	"google.golang.org/grpc/naming"
)

func TestStaticResolver(t *testing.T) {
	addr := []string{"node1:1000", "node2:2000"}
	r := NewStaticResolver(addr...)
	w, err := r.Resolve("")
	if err != nil {
		t.Fatal(err)
	}

	w.Close() // doesn't do anything, and cause no harm

	updates, err := w.Next()
	if err != nil {
		t.Fatal(err)
	}
	if want, have := len(addr), len(updates); want != have {
		t.Fatalf("retrieve updates via Next(): want %d, have %d", want, have)
	}
	if want, have := addr[0], updates[0].Addr; want != have {
		t.Fatalf("1st update Addr: want %q, have %q", want, have)
	}
	if want, have := naming.Add, updates[0].Op; want != have {
		t.Fatalf("1st update Op: want %v, have %v", want, have)
	}
	if want, have := addr[1], updates[1].Addr; want != have {
		t.Fatalf("2nd update Addr: want %q, have %q", want, have)
	}
	if want, have := naming.Add, updates[1].Op; want != have {
		t.Fatalf("2nd update Op: want %v, have %v", want, have)
	}

	// Further calls to w.Next should block
	res := make(chan struct{}, 1)
	go func() {
		_, err := w.Next()
		if err != nil {
			t.Fatal(err)
		}
		res <- struct{}{}
	}()
	select {
	case <-res:
		t.Fatal("further calls to Next() should block")
	case <-time.After(250 * time.Millisecond):
	}
}
