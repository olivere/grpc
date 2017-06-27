// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package static

import (
	"google.golang.org/grpc/naming"
)

// Resolver implements a gRPC resolver/watcher that simply returns
// a list of addresses, then blocks.
type Resolver struct {
	addr []*naming.Update
}

// NewResolver initializes and returns a new Resolver.
func NewResolver(addr ...string) *Resolver {
	r := &Resolver{}
	for _, a := range addr {
		r.addr = append(r.addr, &naming.Update{Op: naming.Add, Addr: a})
	}
	return r
}

// Resolve creates a watcher for target. The watcher interface is implemented
// by Resolver as well, see Next and Close.
func (r *Resolver) Resolve(target string) (naming.Watcher, error) {
	return r, nil
}

// Next returns the list of addresses once, then blocks on consecutive calls.
func (r *Resolver) Next() ([]*naming.Update, error) {
	if r.addr != nil {
		updates := r.addr
		r.addr = nil
		return updates, nil
	}
	infinite := make(chan struct{})
	<-infinite
	return nil, nil
}

// Close is a no-op for a Resolver.
func (r *Resolver) Close() {}
