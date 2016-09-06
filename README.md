# gRPC packages and tools

This repository contains packages and tools for [gRPC](http://www.grpc.io/)
for the Go programming language.

## Load Balancing

The [`github.com/olivere/grpc/lb` package](https://github.com/olivere/grpc/blob/master/lb) implements load-balancing as described in [this document](https://github.com/grpc/grpc/blob/master/doc/load-balancing.md).

It currently supports a Consul-based load balancer plus a simple load-balancer
based on a static list of addresses.

## License

MIT-LICENSE. See [LICENSE](http://olivere.mit-license.org/)
or the LICENSE file provided in the repository for details.


This package implements gRPC load-balancing as described
in [this document](https://github.com/grpc/grpc/blob/master/doc/load-balancing.md).

It has two `Resolver` implementations:
* [StaticResolver]()
* [ConsulResolver]()

Here's an example of setting up a Consul-based resolver for a gRPC client:
