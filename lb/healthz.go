// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package lb

import (
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/naming"
)

var (
	healthzDefaultCheckTimeout   = 5 * time.Second
	healthzDefaultUpdateInterval = 30 * time.Second
)

// HealthzResolver implements the gRPC Resolver interface using a simple
// health endpoint check on a list of clients initially passed to the
// resolver.
//
// See the gRPC load balancing documentation for details about Balancer and
// Resolver: https://github.com/grpc/grpc/blob/master/doc/load-balancing.md.
type HealthzResolver struct {
	mu   sync.Mutex
	endp []*HealthzEndpoint

	checkTimeout   time.Duration
	updateInterval time.Duration

	quitc    chan struct{}
	updatesc chan []*naming.Update
}

// HealthzEndpoint is an endpoint that serves gRPC and responds to health
// checks on the CheckURL.
type HealthzEndpoint struct {
	Addr     string // e.g. 127.0.0.1:10000
	CheckURL string // e.g. http://127.0.0.1:10000/healthz

	status int // last HTTP status for CheckURL
}

// NewHealthzResolver initializes and returns a new HealthzResolver.
//
// It resolves addresses for gRPC connections to the given list of host:port
// endpoints. It does
func NewHealthzResolver(endpoints ...HealthzEndpoint) (*HealthzResolver, error) {
	endp := make([]*HealthzEndpoint, len(endpoints))
	for i, ep := range endpoints {
		endp[i] = &HealthzEndpoint{
			Addr:     ep.Addr,
			CheckURL: ep.CheckURL,
			status:   http.StatusNotImplemented,
		}
	}
	r := &HealthzResolver{
		endp:           endp,
		checkTimeout:   healthzDefaultCheckTimeout,
		updateInterval: healthzDefaultUpdateInterval,
		quitc:          make(chan struct{}),
		updatesc:       make(chan []*naming.Update, len(endp)),
	}

	// Run an initial update to ensure the endpoints are valid.
	updates, err := r.update()
	if err != nil {
		return nil, err
	}
	r.updatesc <- updates

	// Start updater
	go r.updater()

	return r, nil
}

// Resolve creates a watcher for target. The watcher interface is implemented
// by HealthzResolver as well, see Next and Close.
func (r *HealthzResolver) Resolve(target string) (naming.Watcher, error) {
	return r, nil
}

// Next blocks until an update or error happens. It may return one or more
// updates. The first call will return the full set of instances available
// as NewHealthzResolver will look those up. Subsequent calls to Next() will
// block until the resolver finds any new or removed instance.
//
// An error is returned if and only if the watcher cannot recover.
func (r *HealthzResolver) Next() ([]*naming.Update, error) {
	return <-r.updatesc, nil
}

// Close closes the watcher.
func (r *HealthzResolver) Close() {
	select {
	case <-r.quitc:
	default:
		close(r.quitc)
		close(r.updatesc)
	}
}

// updater is a background process started in NewHealthzResolver.
func (r *HealthzResolver) updater() {
	t := time.NewTicker(r.updateInterval)
	defer t.Stop()

	for {
		select {
		case <-r.quitc:
			break
		case <-t.C:
			updates, err := r.update()
			if err != nil {
				log.Printf("grpc/lb: error retrieving updates: %v", err)
				continue
			}
			r.updatesc <- updates
		}
	}
}

// update checks the endpoints, sets their alive flag and returns a list
// of updates in an array of naming.Updates.
func (r *HealthzResolver) update() ([]*naming.Update, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldStatuses := make(map[*HealthzEndpoint]int)
	for _, ep := range r.endp {
		oldStatuses[ep] = ep.status
	}

	// Run all checks in parallel
	ctx, cancel := context.WithTimeout(context.Background(), r.checkTimeout)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	for _, ep := range r.endp {
		ep := ep // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			res, err := ctxhttp.Get(ctx, http.DefaultClient, ep.CheckURL)
			if err != nil {
				return err
			}
			defer res.Body.Close()
			ep.status = res.StatusCode
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var updates []*naming.Update
	for ep, oldStatus := range oldStatuses {
		// fmt.Printf("%v changed from %d to %d\n", ep.Addr, oldStatus, ep.status)
		oldOK := oldStatus >= 200 && oldStatus < 300
		newOK := ep.status >= 200 && ep.status < 300
		if oldOK && !newOK {
			// Was OK, is no longer OK => Delete
			updates = append(updates, &naming.Update{Op: naming.Delete, Addr: ep.Addr})
		} else if !oldOK && newOK {
			// Has failed, is OK now => Add
			updates = append(updates, &naming.Update{Op: naming.Add, Addr: ep.Addr})
		}
	}
	return updates, nil
}
