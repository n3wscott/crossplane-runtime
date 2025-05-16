package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

type server struct {
	v1alpha1.UnimplementedExternalServiceServer
}

func (s *server) Session(stream v1alpha1.ExternalService_SessionServer) error {
	clientID := []byte(uuid.NewUUID())
	fmt.Printf("Starting Operation Instance %s\n", clientID)
	for {
		// Read request from stream
		req, err := stream.Recv()
		if err != nil {
			return err
		}

		switch op := req.Op.(type) {
		case *v1alpha1.Request_Connect:
			fmt.Printf("Connect!! %#v\n", op.Connect)
			resp := &v1alpha1.ConnectResponse{
				Resource:          op.Connect.Resource,
				ConnectionDetails: map[string][]byte{"Client": clientID},
			}
			if err := stream.Send(&v1alpha1.Response{Op: &v1alpha1.Response_Connect{Connect: resp}}); err != nil {
				fmt.Println(err)
				return err
			}
		case *v1alpha1.Request_Observe:
			fmt.Printf("Observe!! %#v\n", op.Observe)
			resp := &v1alpha1.ObserveResponse{
				Resource:                op.Observe.Resource,
				ConnectionDetails:       map[string][]byte{"Client": clientID},
				ResourceExists:          true,
				ResourceUpToDate:        true,
				ResourceLateInitialized: true,
			}
			if err := stream.Send(&v1alpha1.Response{Op: &v1alpha1.Response_Observe{Observe: resp}}); err != nil {
				fmt.Println(err)
				return err
			}
		case *v1alpha1.Request_Create:
			fmt.Printf("Create!! %#v\n", op.Create)
			resp := &v1alpha1.CreateResponse{
				Resource:          op.Create.Resource,
				ConnectionDetails: map[string][]byte{"Client": clientID},
				AdditionalDetails: map[string]string{"more": "call me back"},
			}
			if err := stream.Send(&v1alpha1.Response{Op: &v1alpha1.Response_Create{Create: resp}}); err != nil {
				fmt.Println(err)
				return err
			}
		case *v1alpha1.Request_Update:
			fmt.Printf("Update!! %#v\n", op.Update)
			resp := &v1alpha1.UpdateResponse{
				Resource:          op.Update.Resource,
				ConnectionDetails: map[string][]byte{"Client": clientID},
				AdditionalDetails: map[string]string{"more": "call me back"},
			}
			if err := stream.Send(&v1alpha1.Response{Op: &v1alpha1.Response_Update{Update: resp}}); err != nil {
				fmt.Println(err)
				return err
			}
		case *v1alpha1.Request_Delete:
			fmt.Printf("Delete!! %#v\n", op.Delete)
			resp := &v1alpha1.DeleteResponse{
				Resource:          op.Delete.Resource,
				AdditionalDetails: map[string]string{"more": "hang up"},
			}
			if err := stream.Send(&v1alpha1.Response{Op: &v1alpha1.Response_Delete{Delete: resp}}); err != nil {
				fmt.Println(err)
				return err
			}
		case *v1alpha1.Request_Disconnect:
			fmt.Printf("Disconnect!! %#v\n", op.Disconnect)
			resp := &v1alpha1.DisconnectResponse{}
			if err := stream.Send(&v1alpha1.Response{Op: &v1alpha1.Response_Disconnect{Disconnect: resp}}); err != nil {
				fmt.Println(err)
				return err
			}
		default:
			fmt.Println("No matching operations")
			return nil
		}
	}
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	v1alpha1.RegisterExternalServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
