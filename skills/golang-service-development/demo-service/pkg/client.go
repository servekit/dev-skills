package pkg

import (
	"context"

	demov1 "demo-service/gen/demo/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Client is a gRPC client for demo-service shaped like *Handler: it implements
// the generated demov1.DemoServiceServer interface (unary methods
// without grpc.CallOption), so a consumer can hold either backend behind the
// provider-defined Service interface — module mode passes the *Handler, grpc
// mode passes the *Client — with no per-consumer adapter.
//
// The UnimplementedDemoServiceServer embed satisfies the interface's
// mustEmbed guard; every RPC below shadows it with a real delegation. When a
// new RPC is added to the proto, add its delegation here — until then grpc
// mode returns codes.Unimplemented for it.
type Client struct {
	demov1.UnimplementedDemoServiceServer

	conn *grpc.ClientConn
	cli  demov1.DemoServiceClient
}

// Compile-time assertion: *Client and *Handler expose the same interface.
var _ demov1.DemoServiceServer = (*Client)(nil)

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
		conn: conn,
		cli:  demov1.NewDemoServiceClient(conn),
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *Client) Close() error { return c.conn.Close() }

// Ping delegates to the remote demo-service.
func (c *Client) Ping(ctx context.Context, in *emptypb.Empty) (*demov1.Pong, error) {
	return c.cli.Ping(ctx, in)
}

// CreateDemo delegates to the remote demo-service.
func (c *Client) CreateDemo(ctx context.Context, in *demov1.CreateDemoRequest) (*demov1.Demo, error) {
	return c.cli.CreateDemo(ctx, in)
}

// GetDemo delegates to the remote demo-service.
func (c *Client) GetDemo(ctx context.Context, in *demov1.GetDemoRequest) (*demov1.Demo, error) {
	return c.cli.GetDemo(ctx, in)
}

// ListDemos delegates to the remote demo-service.
func (c *Client) ListDemos(ctx context.Context, in *demov1.ListDemosRequest) (*demov1.ListDemosResponse, error) {
	return c.cli.ListDemos(ctx, in)
}

// UpdateDemo delegates to the remote demo-service.
func (c *Client) UpdateDemo(ctx context.Context, in *demov1.UpdateDemoRequest) (*demov1.Demo, error) {
	return c.cli.UpdateDemo(ctx, in)
}

// DeleteDemo delegates to the remote demo-service.
func (c *Client) DeleteDemo(ctx context.Context, in *demov1.DeleteDemoRequest) (*emptypb.Empty, error) {
	return c.cli.DeleteDemo(ctx, in)
}
