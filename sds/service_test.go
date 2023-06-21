package sds_test

import (
	"context"
	"fmt"
	"net"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/kanurag94/envoy-sds/sds"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestService_FetchSecrets(t *testing.T) {
	srv := sds.New()
	defer srv.Stop()

	// Prepare server
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	srv.Register(s)
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(fmt.Sprintf("Server exited with error: %v", err))
		}
	}()

	// Prepare client
	ctx := context.Background()
	dialer := func(ctx context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(ctx, "bufconn", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufconn: %v", err)
	}
	defer conn.Close()
	client := secret.NewSecretDiscoveryServiceClient(conn)

	tests := []struct {
		name    string
		req     *discovery.DiscoveryRequest
		wantErr bool
	}{
		{"ok", &discovery.DiscoveryRequest{
			VersionInfo:   "versionInfo",
			Node:          &corev3.Node{Id: "node-id", Cluster: "node-cluster"},
			ResourceNames: []string{"one"},
			TypeUrl:       "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret",
			ResponseNonce: "response-nonce",
		}, false},
		{"ok multiple", &discovery.DiscoveryRequest{
			VersionInfo:   "versionInfo",
			Node:          &corev3.Node{Id: "node-id", Cluster: "node-cluster"},
			ResourceNames: []string{"one", "two"},
			TypeUrl:       "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret",
			ResponseNonce: "response-nonce",
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := client.FetchSecrets(context.Background(), tt.req)
			fmt.Println(c)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.FetchSecrets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
