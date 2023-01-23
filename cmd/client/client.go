package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	pb "github.com/medenzel/grpc-fileserver/proto"
	log "github.com/sirupsen/logrus"
)

// Client - grpc client object
type Client struct {
	Proto pb.FileServiceClient
	Root  string
}

// NewClient - client constructor
func NewClient(protoClient pb.FileServiceClient, root string) *Client {
	return &Client{Proto: protoClient, Root: root}
}

// Download - transfers files from server to client
func (c *Client) Download(filename string) error {
	log.Printf("Downloading file: %s", filename)
	stream, err := c.Proto.Download(context.Background(), &pb.DownloadRequest{Filename: filename})
	if err != nil {
		log.Errorf("Error downloading: %v", err)
		return err
	}
	file, err := os.Create(filepath.Join(c.Root, filename))
	if err != nil {
		log.Errorf("error creating file: %v", err)
		return err
	}
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("error receiving chunk: %v", err)
			file.Close()
			errRm := os.Remove(filepath.Join(c.Root, filename))
			if errRm == nil {
				return err
			}
			return errRm
		}
		_, err = file.Write(chunk.Filechank)
		if err != nil {
			log.Errorf("error writing to file: %v", err)
			return err
		}
	}
	log.Printf("Downloaded successfully")
	file.Close()
	return nil
}

// Upload - transfer files from client to server
func (c *Client) Upload(filename string) error {
	stream, err := c.Proto.Upload(context.Background())
	if err != nil {
		log.Errorf("error upload: %v", err)
		return err
	}
	err = stream.Send(&pb.UploadRequest{Data: &pb.UploadRequest_Filename{Filename: filename}})
	if err != nil {
		log.Errorf("error sending request: %v", err)
		return err
	}
	f, err := os.Open(filepath.Join(c.Root, filename))
	if err != nil {
		log.Errorf("Error opening file: %v", err)
		return err
	}
	defer f.Close()
	buf := make([]byte, 1024)
	filesize := 0
	for {
		n, err := f.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Errorf("Error reading file: %v", err)
				return err
			}
			log.Print("File successfully read")
			break
		}
		filesize += n
		err = stream.Send(&pb.UploadRequest{
			Data: &pb.UploadRequest_Filechank{
				Filechank: buf[:n]}})
		if err != nil {
			log.Errorf("Error sending chunk: %v", err)
			return err
		}
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Errorf("Error getting response: %v", err)
		return err
	}
	if int32(filesize) == resp.GetSize() {
		return errors.New("Error occured: readed bytes doesn't equal sent bytes!")
	}
	return nil
}

// List - receives all available files in the server storage
func (c *Client) List() error {
	stream, err := c.Proto.ListFiles(context.Background(), &pb.ListRequest{})
	if err != nil {
		log.Errorf("error listing files: %v", err)
		return err
	}
	fmt.Printf("%30s | %16s | %30s \n", "Filename", "Size", "Modification time")
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("Error receiving files: %v", err)
			return err
		}
		fmt.Printf("%30s | %10d bytes | %30v \n", resp.GetFilename(), resp.GetSize(), resp.GetModTime().AsTime())
	}
	log.Print("All files listed!")
	return nil
}
