package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	pb "github.com/medenzel/grpc-fileserver/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	ts "google.golang.org/protobuf/types/known/timestamppb"
)

// Server - grpc server object
type Server struct {
	pb.FileServiceServer
	Root string
}

// Download - transfers files from server to client
func (s *Server) Download(req *pb.DownloadRequest, stream pb.FileService_DownloadServer) error {
	log.Print("Download request received")
	file, err := os.Open(filepath.Join(s.Root, req.GetFilename()))
	if err != nil {
		log.Errorf("Error opening file: %v", err)
		return status.Errorf(codes.Internal, fmt.Sprintf("error opening file: %v", err))
	}
	defer file.Close()
	bufferSize := 1024
	buff := make([]byte, bufferSize)
	for {
		n, err := file.Read(buff)
		if err != nil {
			if err != io.EOF {
				log.Errorf("Error reading file: %v", err)
				return status.Errorf(codes.Internal, fmt.Sprintf("error reading file: %v", err))
			}
			break
		}
		err = stream.Send(&pb.DownloadResponse{
			Filechank: buff[:n],
		})
		if err != nil {
			log.Errorf("Error sending: %v", err)
			return status.Errorf(codes.Internal, fmt.Sprintf("error sending: %v", err))
		}
	}
	log.Print("Successfully send!")
	return nil
}

// Upload - transfers files from client to server
func (s *Server) Upload(stream pb.FileService_UploadServer) error {
	log.Print("Upload request received")
	req, err := stream.Recv()
	if err != nil {
		log.Errorf("Error receiving: %v", err)
		return status.Error(codes.Unknown, fmt.Sprint("Error receiving"))
	}
	filename := req.GetFilename()
	file, err := os.Create(filepath.Join(s.Root, filename))
	if err != nil {
		log.Errorf("Error creating file: %v", err)
		return status.Error(codes.Internal, fmt.Sprintf("Error creating file: %v", err))
	}
	filesize := 0
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("EOF received")
			break
		}
		if err != nil {
			log.Errorf("Error receiving chunks: %v", err)
			file.Close()
			errRm := os.Remove(filepath.Join(s.Root, filename))
			if errRm == nil {
				return status.Error(codes.Internal, fmt.Sprintf("error receiving filechunk: %v", err))
			}
			return errRm
		}
		n, err := file.Write(req.GetFilechank())
		if err != nil {
			log.Errorf("Error writing chunk: %v", err)
			return status.Error(codes.Internal, fmt.Sprint("error writing to file"))
		}
		filesize += n
	}
	err = stream.SendAndClose(&pb.UploadResponse{Size: int32(filesize)})
	if err != nil {
		log.Errorf("Error sending response: %v", err)
		return status.Error(codes.Internal, fmt.Sprintf("error send response: %v", err))
	}
	file.Close()
	log.Print("Successfully uploaded!")
	return nil
}

// ListFiles - sends all files (available in server root directory)
// info (name/size/modification time) to the client
func (s *Server) ListFiles(req *pb.ListRequest, stream pb.FileService_ListFilesServer) error {
	log.Print("List request received")
	err := filepath.WalkDir(s.Root, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		name, err := filepath.Rel(s.Root, path)
		if err != nil {
			log.Errorf("Error listfiles rel: %v", err)
			return status.Error(codes.Internal, fmt.Sprint("error walking dir"))
		}
		fileInfo, err := d.Info()
		if err != nil {
			log.Errorf("Error listfiles fileinfo: %v", err)
			return status.Error(codes.Internal, fmt.Sprint("error walking dir"))
		}
		size := fileInfo.Size()
		modTime := ts.New(fileInfo.ModTime())
		err = stream.Send(&pb.ListResponse{
			Filename: name,
			Size:     size,
			ModTime:  modTime})
		if err != nil {
			log.Errorf("Error sending response: %v", err)
			return status.Error(codes.Unknown, fmt.Sprintf("error sending response: %v", err))
		}
		return nil
	})
	if err != nil {
		log.Errorf("Error walking dir: %v", err)
		return status.Error(codes.Unknown, fmt.Sprintf("error getting list: %v", err))
	}
	log.Printf("Successfully listed!")
	return nil
}
