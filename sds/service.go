package sds

import (
	"context"
	"errors"
	"fmt"
	"time"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	secret "github.com/envoyproxy/go-control-plane/envoy/service/secret/v3"
	"google.golang.org/grpc"
)

var Identifier = "envoy-sds"

type Service struct {
	stopCh chan struct{}
}

func New() *Service {
	return &Service{
		stopCh: make(chan struct{}),
	}
}

// Stop stops the current service.
func (srv *Service) Stop() error {
	close(srv.stopCh)
	return nil
}

// Register registers the sds.Service into the given gRPC server.
func (srv *Service) Register(s *grpc.Server) {
	secret.RegisterSecretDiscoveryServiceServer(s, srv)
}

func (srv *Service) DeltaSecrets(sds secret.SecretDiscoveryService_DeltaSecretsServer) (err error) {
	return errors.New("method DeltaSecrets not implemented")
}

// StreamSecrets implements the gRPC SecretDiscoveryService service and returns
// a stream of envoy generic secrets.
func (srv *Service) StreamSecrets(sds secret.SecretDiscoveryService_StreamSecretsServer) (err error) {
	// ctx := sds.Context()
	errCh := make(chan error)
	reqCh := make(chan *discovery.DiscoveryRequest)

	go func() {
		for {
			r, err := sds.Recv()
			if err != nil {
				errCh <- err
				return
			}
			reqCh <- r
		}
	}()

	var nonce, versionInfo string
	var req *discovery.DiscoveryRequest

	for {
		select {
		case r := <-reqCh:
			if r.ErrorDetail != nil {
				fmt.Printf("failed discovery request, error: %s", err.Error())
				continue
			}

			// Do not validate nonce/version if we're restarting the server
			if req != nil {
				switch {
				case nonce != r.ResponseNonce:
					fmt.Printf("invalid responseNonce")
					continue
				case r.VersionInfo == "": // first request
					versionInfo = srv.versionInfo()
				case r.VersionInfo == versionInfo: // consecutive request ACK
					fmt.Println("Receieved ACK")
					continue
				default:
					versionInfo = srv.versionInfo()
				}
			} else {
				versionInfo = srv.versionInfo()
			}

			req = r
			for _, name := range req.ResourceNames {
				fmt.Printf("Request for resource: %s received", name)
			}

		case err := <-errCh:
			fmt.Printf("error occured on channel %s", err.Error())
			return err
		case <-srv.stopCh:
			return nil
		}

		// Send secrets
		dr, err := getDiscoveryResponse(req, versionInfo)
		if err != nil {
			fmt.Printf("error while creating response: %s", err.Error())
			return err
		}
		if err := sds.Send(dr); err != nil {
			fmt.Printf("error sending stream response: %s", err.Error())
			return err
		}

		nonce = dr.Nonce
	}
}

// FetchSecrets implements gRPC SecretDiscoveryService service and returns one TLS certificate.
func (srv *Service) FetchSecrets(ctx context.Context, r *discovery.DiscoveryRequest) (*discovery.DiscoveryResponse, error) {
	return getDiscoveryResponse(r, srv.versionInfo())
}

func (srv *Service) versionInfo() string {
	return time.Now().UTC().Format(time.RFC3339)
}
