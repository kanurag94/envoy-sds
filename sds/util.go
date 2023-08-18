package sds

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const secretTypeURL = "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.Secret"

// getDiscoveryResponse returns the api.DiscoveryResponse for the given request.
func getDiscoveryResponse(r *discovery.DiscoveryRequest, versionInfo string) (*discovery.DiscoveryResponse, error) {
	nonce, err := randomHex(64)
	if err != nil {
		return nil, fmt.Errorf("error generating nonce: %s", err.Error())
	}

	var b []byte
	var resources []*anypb.Any
	for _, name := range r.ResourceNames {
		b, err = getGenericSecret(name)
		if err != nil {
			return nil, err
		}
		resources = append(resources, &anypb.Any{
			TypeUrl: secretTypeURL,
			Value:   b,
		})
	}

	return &discovery.DiscoveryResponse{
		VersionInfo: versionInfo,
		Resources:   resources,
		TypeUrl:     secretTypeURL,
		Nonce:       nonce,
	}, nil
}

func getGenericSecret(name string) ([]byte, error) {
	secret := auth.Secret{
		Name: name,
		Type: &auth.Secret_GenericSecret{
			GenericSecret: &auth.GenericSecret{
				Secret: &corev3.DataSource{
					Specifier: &corev3.DataSource_InlineBytes{
						InlineBytes: make([]byte, 32),
					},
				},
			},
		},
	}
	v, err := proto.Marshal(&secret)
	if err != nil {
		return v, fmt.Errorf("error marshaling secret: %s", err.Error())
	}
	return v, err
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
