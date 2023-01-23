package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/medenzel/grpc-fileserver/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	addrFlag = flag.String("addr", "0.0.0.0", "Set server address")
	portFlag = flag.String("p", "50051", "Set server port")
	dirFlag  = flag.String("dir", ".", "Set directory for file storage")
)

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	flag.Parse()

	addr := *addrFlag + ":" + *portFlag

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer(grpc.ConnectionTimeout(1 * time.Minute))
	pb.RegisterFileServiceServer(s, &Server{Root: *dirFlag})

	log.Printf("Listening on addr: %s", addr)
	log.Printf("File folder: %s", *dirFlag)

	go func() {
		if err = s.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	s.GracefulStop()
	log.Print("Gracefully shutdowned.")
}
