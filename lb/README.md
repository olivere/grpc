# gRPC Load Balancing in 5 minutes

This package implements gRPC load-balancing as described
in [this document](https://github.com/grpc/grpc/blob/master/doc/load-balancing.md).

It has these `Resolver` implementations:
* [StaticResolver](static/static.go)
* [HealthzResolver](static/static.go)
* [ConsulResolver](static/static.go)

Here's an example of setting up a Consul-based resolver for a gRPC client:

```go
import (
	"github.com/hashicorp/consul/api"
)

func main() {
	// Create Consul client
	cli, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

  // Create a resolver for the "echo" service
	r, err := lb.NewConsulResolver(cli, "echo", "")
	if err != nil {
		log.Fatal(err)
	}

	// Setup a gRPC client connection
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBalancer(grpc.RoundRobin(r)))

  // Notice you can use a blank address here
	conn, err := grpc.Dial("", opts...)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Every call to conn will get load-balanced between the servers
	// found for the "echo" service in Consul, e.g.:
	for i := 0; i < 100; i++ {
		ctx := context.Background()
		res, err := client.Echo(ctx, &pb.EchoRequest{Message: "Hello"})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%v\n", res.Message)
		time.Sleep(1*time.Second)
	}
}

```

You don't need to do anything for the gRPC server-side (except registering
your service in Consul, of course).

See the [examples]() directory for a working gRPC client/server implementation.
