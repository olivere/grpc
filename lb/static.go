// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package lb

import (
	// "google.golang.org/grpc"
	"google.golang.org/grpc/naming"
)

type StaticResolver struct {
	addr []*naming.Update
}

func NewStaticResolver(addr ...string) *StaticResolver {
	r := &StaticResolver{}
	for _, a := range addr {
		r.addr = append(r.addr, &naming.Update{Op: naming.Add, Addr: a})
	}
	return r
}

func (r *StaticResolver) Resolve(target string) (naming.Watcher, error) {
	return r, nil
}

func (r *StaticResolver) Next() ([]*naming.Update, error) {
	if r.addr != nil {
		updates := r.addr
		r.addr = nil
		return updates, nil
	}
	infinite := make(chan struct{})
	<-infinite
	return nil, nil
}

func (r *StaticResolver) Close() {}
