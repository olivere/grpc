// Copyright 2016-present Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package healthz

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/naming"
)

var (
	defaultCheckTimeout   = 5 * time.Second
	defaultUpdateInterval = 30 * time.Second

	// ErrNoEndpoints is returned when you passed no endpoints to the Resolver.
	ErrNoEndpoints = errors.New("no endpoints specified")
)

// Logger allows to pass an optional logger to the resolver.
type Logger interface {
	Printf(format string, values ...interface{})
}

// nopLogger implements Logger but does not log.
type nopLogger struct{}

// Printf does not log anything.
func (nopLogger) Printf(format string, v ...interface{}) {}

// Resolver implements the gRPC Resolver interface using a simple
// health endpoint check on a list of clients initially passed to the
// resolver.
//
// See the gRPC load balancing documentation for details about Balancer and
// Resolver: https://github.com/grpc/grpc/blob/master/doc/load-balancing.md.
type Resolver struct {
	mu   sync.Mutex
	endp []*Endpoint

	logger         Logger
	checkTimeout   time.Duration
	updateInterval time.Duration

	quitc    chan struct{}
	updatesc chan []*naming.Update
}

// Endpoint is an endpoint that serves gRPC and responds to health
// checks on the CheckURL.
type Endpoint struct {
	Addr     string // e.g. 127.0.0.1:10000
	CheckURL string // e.g. http://127.0.0.1:10000/healthz

	status int // last HTTP status for CheckURL
}

// ResolverOption is a callback for setting the options of the Resolver.
type ResolverOption func(*Resolver) error

// NewResolver initializes and returns a new Resolver.
//
// It resolves addresses for gRPC connections to the given list of host:port
// endpoints. It runs HTTP-based health checks periodically to ensure that
// all endpoints are still reachable and healthy. If an endpoint does not
// respond in time, it is removed from the list of valid endpoints. Once it
// comes up again, it will be added to the list of healthy endpoints again,
// and traffic will be served to that endpoint again.
func NewResolver(options ...ResolverOption) (*Resolver, error) {
	r := &Resolver{
		logger:         nopLogger{},
		checkTimeout:   defaultCheckTimeout,
		updateInterval: defaultUpdateInterval,
		quitc:          make(chan struct{}),
	}
	for _, option := range options {
		if err := option(r); err != nil {
			return nil, err
		}
	}
	if len(r.endp) == 0 {
		return nil, ErrNoEndpoints
	}
	r.updatesc = make(chan []*naming.Update, len(r.endp))

	// Run an initial update to ensure the endpoints are valid on the first call.
	// Don't worry if there are no healthy endpoints, just continue to watch.
	updates, err := r.update()
	if err == nil && len(updates) > 0 {
		r.updatesc <- updates
	}

	// Start updater
	go r.updater()

	return r, nil
}

// SetEndpoints specifies the endpoints for the resolver.
func SetEndpoints(endpoints ...Endpoint) ResolverOption {
	return func(r *Resolver) error {
		endp := make([]*Endpoint, len(endpoints))
		for i, ep := range endpoints {
			endp[i] = &Endpoint{
				Addr:     ep.Addr,
				CheckURL: ep.CheckURL,
				status:   http.StatusServiceUnavailable,
			}
		}
		r.endp = endp
		return nil
	}
}

// SetLogger allows to pass a logger for Resolver.
func SetLogger(logger Logger) ResolverOption {
	return func(r *Resolver) error {
		r.logger = logger
		return nil
	}
}

// SetCheckTimeout specifies the duration after which an endpoint
// is considered gone in a health check.
func SetCheckTimeout(timeout time.Duration) ResolverOption {
	return func(r *Resolver) error {
		r.checkTimeout = timeout
		return nil
	}
}

// SetUpdateInterval specifies the interval in which to run health checks.
func SetUpdateInterval(interval time.Duration) ResolverOption {
	return func(r *Resolver) error {
		r.updateInterval = interval
		return nil
	}
}

// Resolve creates a watcher for target. The watcher interface is implemented
// by Resolver as well, see Next and Close.
func (r *Resolver) Resolve(target string) (naming.Watcher, error) {
	return r, nil
}

// Next blocks until an update or error happens. It may return one or more
// updates. The first call will return the full set of instances available
// as NewResolver will look those up. Subsequent calls to Next() will
// block until the resolver finds any new or removed instance.
//
// An error is returned if and only if the watcher cannot recover.
func (r *Resolver) Next() ([]*naming.Update, error) {
	return <-r.updatesc, nil
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

// updater is a background process started in NewResolver.
func (r *Resolver) updater() {
	t := time.NewTicker(r.updateInterval)
	defer t.Stop()

	for {
		select {
		case <-r.quitc:
			break
		case <-t.C:
			updates, err := r.update()
			if err != nil {
				r.logger.Printf("grpc/lb/healthz: error retrieving updates: %v", err)
				continue
			}
			r.updatesc <- updates
		}
	}
}

// update checks the endpoints, sets their alive flag and returns a list
// of updates in an array of naming.Updates.
func (r *Resolver) update() ([]*naming.Update, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldStatuses := make(map[*Endpoint]int)
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
				// Mark endpoint as unhealthy
				ep.status = http.StatusServiceUnavailable
				return nil
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
