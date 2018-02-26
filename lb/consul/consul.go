// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package consul

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"google.golang.org/grpc/naming"
)

// NewStreamDemux initializes and returns a new StreamDemux
func NewStreamDemux(c *api.Client) *StreamDemux {
	sd := &StreamDemux{
		c:        c,
		services: make(map[string][](chan []string)),
	}
	go func() {
		for {
			time.Sleep(time.Second * 5)
			sd.getInstances(0)
		}
	}()

	return sd
}

// StreamDemux provides the list of list of available serves via a Consul backend.
type StreamDemux struct {
	c        *api.Client
	services map[string][](chan []string)
}

// getInstances retrieves all instances of the services registered in Consul.
func (sd StreamDemux) getInstances(lastIndex uint64) {
	fmt.Println("ok papa get some consul instances")

	var addresses []string

	for s, scs := range sd.services {
		services, _, _ := sd.c.Health().Service(s, "", true, &api.QueryOptions{
			WaitIndex: lastIndex,
		})

		for _, service := range services {
			s := service.Service.Address
			if len(s) == 0 {
				s = service.Node.Address
			}
			addr := net.JoinHostPort(s, strconv.Itoa(service.Service.Port))
			addresses = append(addresses, addr)
		}
		fmt.Printf("%s: %v \n", s, addresses)
		for _, sc := range scs {
			sc <- addresses
		}
	}
}

// NewResolver initializes and returns a new Resolver.
func (sd StreamDemux) NewResolver(service, tag string) (*Resolver, error) {
	resolverSc := make(chan []string)
	sd.services[service] = append(sd.services[service], resolverSc)
	r := &Resolver{
		service:   service,
		tag:       tag,
		instances: nil,
		servicesc: resolverSc,
		quitc:     make(chan struct{}),
		updatesc:  make(chan []*naming.Update, 1),
	}

	go r.makeUpdates()

	return r, nil
}

// Resolver implements the gRPC Resolver interface using a Consul backend.
//
// See the gRPC load balancing documentation for details about Balancer and
// Resolver: https://github.com/grpc/grpc/blob/master/doc/load-balancing.md.
type Resolver struct {
	service   string
	tag       string
	instances []string

	servicesc chan []string
	quitc     chan struct{}
	updatesc  chan []*naming.Update
}

// Resolve creates a watcher for target. The watcher interface is implemented
// by Resolver as well, see Next and Close.
func (r *Resolver) Resolve(target string) (naming.Watcher, error) {
	fmt.Println("target", target)
	return r, nil
}

// Next blocks until an update or error happens. It may return one or more
// updates. The first call will return the full set of instances available
// as NewResolver will look those up. Subsequent calls to Next() will
// block until the resolver finds any new or removed instance.
//
// An error is returned if and only if the watcher cannot recover.
func (r *Resolver) Next() ([]*naming.Update, error) {
	nextUpdate := <-r.updatesc
	for _, v := range nextUpdate {
		fmt.Printf("%v \n", v)
	}
	return nextUpdate, nil
}

// Close closes the watcher.
func (r *Resolver) Close() {
	select {
	case <-r.quitc:
	default:
		close(r.quitc)
		close(r.updatesc)
	}
}

// makeUpdates calculates the difference between and old and a new set of
// instances and turns it into an array of naming.Updates.
func (r *Resolver) makeUpdates() {
	for newInstances := range r.servicesc {
		fmt.Printf("newInstances: %v \n", newInstances)
		oldAddr := make(map[string]struct{}, len(r.instances))
		for _, instance := range r.instances {
			oldAddr[instance] = struct{}{}
		}
		newAddr := make(map[string]struct{}, len(newInstances))
		for _, instance := range newInstances {
			newAddr[instance] = struct{}{}
		}

		var updates []*naming.Update
		for addr := range newAddr {
			if _, ok := oldAddr[addr]; !ok {
				updates = append(updates, &naming.Update{Op: naming.Add, Addr: addr})
			}
		}
		for addr := range oldAddr {
			if _, ok := newAddr[addr]; !ok {
				updates = append(updates, &naming.Update{Op: naming.Delete, Addr: addr})
			}
		}

		if len(updates) > 0 {
			r.updatesc <- updates
		}
		r.instances = newInstances
	}
}
