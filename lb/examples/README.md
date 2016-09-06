# gRPC Load Balancing in 5 minutes

## Prerequisites

Install [Consul](https://www.consul.io/). You also need the Go toolchain
installed, including gRPC bindings.

## Install

Install and compile the examples:

```sh
$ go get github.com/olivere/grpc
$ cd $GOPATH/src/github.com/olivere/grpc/lb/examples
$ make
```

You should now have two files in the `./bin` directory: `server` and `client`.

## Try it

Run Consul. Something like this should do the job:

```sh
$ consul agent -dev -advertise=127.0.0.1
```

Start some servers (at least 2).

```sh
$ ./bin/server
```

Run the client, and execute e.g. 100 requests:

```sh
$ ./bin/client -n=100
```

Watch how the requests get load-balanced between the servers.
