package pkg

import (
	demov1 "demo-service/gen/demo/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a gRPC client for demo-service.
//
// Embeds demov1.DemoServiceClient so callers can invoke RPCs directly:
//
//	c, _ := pkg.NewClient("localhost:9000")
//	demo, _ := c.GetDemo(ctx, &demov1.GetDemoRequest{Id: 1})
type Client struct {
	conn *grpc.ClientConn
	demov1.DemoServiceClient
}

// NewClient dials demo-service at addr using insecure credentials by default.
// Pass additional DialOptions (e.g., credentials) to override.
func NewClient(addr string, opts ...grpc.DialOption) (*Client, error) {
	dialOpts := append([]grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}, opts...)

	conn, err := grpc.NewClient(addr, dialOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:              conn,
		DemoServiceClient: demov1.NewDemoServiceClient(conn),
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *Client) Close() error { return c.conn.Close() }
