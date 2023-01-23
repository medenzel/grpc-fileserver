package main

import (
	"flag"
	"os"

	pb "github.com/medenzel/grpc-fileserver/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	flagAddr = flag.String("addr", "0.0.0.0", "Set client address")
	flagPort = flag.String("p", "50051", "Set client port")
	flagDir  = flag.String("dir", ".", "Set directory for file storage")
)

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	flag.Parse()

	addr := *flagAddr + ":" + *flagPort

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error connecting to %s", addr)
	}
	defer conn.Close()

	pClient := pb.NewFileServiceClient(conn)
	c := NewClient(pClient, *flagDir)
	c.List()
}
