package main

import (
	"context"
	"log"
	"net"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	PROTOCOL = "unix"
	SOCKET   = "/tmp/uds_path"
)

func main() {
	dialer := func(addr string, t time.Duration) (net.Conn, error) {
		return net.Dial(PROTOCOL, addr)
	}

	conn, err := grpc.Dial(SOCKET, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDialer(dialer))
	if err != nil {
		log.Fatal(err)
	}

	client := secret.NewSecretDiscoveryServiceClient(conn)
	// _, _ = client.FetchSecrets(context.Background(), &discovery.DiscoveryRequest{
	// 	VersionInfo:   "versionInfo",
	// 	Node:          &corev3.Node{Id: "node-id", Cluster: "node-cluster"},
	// 	ResourceNames: []string{"one"},
	// 	TypeUrl:       "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret",
	// 	ResponseNonce: "response-nonce",
	// })

	dReq := []*discovery.DiscoveryRequest{
		{
			VersionInfo:   "versionInfo",
			Node:          &corev3.Node{Id: "node-id", Cluster: "node-cluster"},
			ResourceNames: []string{"one"},
			TypeUrl:       "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret",
			ResponseNonce: "response-nonce",
		},
	}

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(c)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.StreamSecrets(ctx)
	if err != nil {
		log.Fatalf("%v.StreamSecrets(_) = _, %v", client, err)
	}
	for _, point := range dReq {
		if err := stream.Send(point); err != nil {
			log.Fatalf("%v.Send(%v) = %v", stream, point, err)
		}
	}
	reply, err := stream.Recv()
	if err != nil {
		log.Fatalf("%v.Recv() got error %v, want %v", stream, err, nil)
	}

	log.Printf("%v", reply)
}
