package gid_service

import (
	"context"
	"fmt"
)

// grpcGID is a placeholder for a real gid-service gRPC client. The scaffold
// ships the wiring shape (NewGRPC + Close returning a GIDService) but leaves
// the actual dial sketched in a comment — enable it against gid-service's
// real client (gidservice.NewClient) when you deploy in grpc mode. Module mode
// (the default) is fully wired via module.go and exercises the real gid-service
// dependency; grpc mode is opt-in via third_party.gid.mode.
type grpcGID struct {
	target string
	// client *gidservice.Client  // uncomment after wiring the real dial below
}

// NewGRPC returns a GIDService bound to a remote gid-service at target. The
// real dial is sketched in the comment — enable it once you deploy gid-service
// as a remote and want grpc mode:
//
//	client, err := gidservice.NewClient(target)
//	if err != nil {
//	    return nil, fmt.Errorf("dial gid-service: %w", err)
//	}
//	return &grpcGID{target: target, client: client}, nil
func NewGRPC(target string) (GIDService, error) {
	if target == "" {
		return nil, fmt.Errorf("target is required for gid grpc mode")
	}
	return &grpcGID{target: target}, nil
}

// NextID is a stub until the real client is wired. Once g.client is set:
//
//	resp, err := g.client.NextID(ctx, &pb.NextIDRequest{})
//	if err != nil {
//	    return 0, err
//	}
//	return resp.Id, nil
func (g *grpcGID) NextID(_ context.Context) (int64, error) {
	return 0, fmt.Errorf("gid grpc mode not implemented in scaffold (would dial %s); use module mode or wire the real client", g.target)
}

// Close releases the underlying gRPC connection. No-op in the stub; once the
// real client is wired, this calls g.client.Close().
func (g *grpcGID) Close() error { return nil }
