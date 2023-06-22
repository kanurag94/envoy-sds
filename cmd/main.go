package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kanurag94/envoy-sds/sds"
	"google.golang.org/grpc"
)

const (
	// unix socket
	PROTOCOL = "unix"
	SOCKET   = "/tmp/uds_path"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

func main() {
	lis, err := net.Listen(PROTOCOL, SOCKET)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Remove(SOCKET)
		os.Exit(1)
	}()

	server := grpc.NewServer()
	sds := sds.New()
	sds.Register(server)

	log.Printf("server listening at %v", lis.Addr())
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
