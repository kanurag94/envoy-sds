package sds_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"github.com/kanurag94/envoy-sds/sds"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestService_FetchSecrets_MTLS(t *testing.T) {
	srv := sds.New()
	defer srv.Stop()

	// Prepare server

	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:50051")
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer(
		grpc.Creds(tlsCredentials),
	)
	srv.Register(s)
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(fmt.Sprintf("Server exited with error: %v", err))
		}
	}()

	// Prepare client
	ctx := context.Background()
	// dialer := func(ctx context.Context, s string) (net.Conn, error) {
	// 	return lis.Dial()
	// }

	tlsClientCredentials, err := loadClientTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", err)
	}

	conn, err := grpc.DialContext(ctx, "example.com:50051", grpc.WithTransportCredentials(tlsClientCredentials))
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

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	/*
		openssl genrsa -out ca.key 2048
		openssl req -new -x509 -days 365 -key ca.key -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=Acme Root CA" -out ca.crt

		openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=*.example.com" -out server.csr
		openssl x509 -req -extfile <(printf "subjectAltName=DNS:example.com,DNS:www.example.com") -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt
	*/

	pemServerCA, err := ioutil.ReadFile("/home/anurag/workspace/playground/ca-client.crt")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("/home/anurag/workspace/playground/server.crt",
		"/home/anurag/workspace/playground/server.key")
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}

	return credentials.NewTLS(config), nil
}

func loadClientTLSCredentials() (credentials.TransportCredentials, error) {
	/*
		openssl genrsa -out ca-client.key 2048
		openssl req -new -x509 -days 365 -key ca-client.key -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=Acme Root CA" -out ca-client.crt

		openssl req -newkey rsa:2048 -nodes -keyout client.key -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=*.client.com" -out client.csr
		openssl x509 -req -extfile <(printf "subjectAltName=DNS:client.com,DNS:www.client.com") -days 365 -in client.csr -CA ca-client.crt -CAkey ca-client.key -CAcreateserial -out client.crt
	*/

	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := ioutil.ReadFile("/home/anurag/workspace/playground/ca.crt")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Load client's certificate and private key
	clientCert, err := tls.LoadX509KeyPair("/home/anurag/workspace/playground/client.crt",
		"/home/anurag/workspace/playground/client.key")
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
	}

	return credentials.NewTLS(config), nil
}
