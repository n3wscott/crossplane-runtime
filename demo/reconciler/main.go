package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/crossplane/crossplane-runtime/apis/proto/external/v1alpha1"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := v1alpha1.NewExternalServiceClient(conn)

	stream, err := client.Operation(context.Background())
	if err != nil {
		log.Fatalf("error calling GetStream: %v", err)
	}

	x := map[string]map[string]string{
		"spec": {
			"example": "ask",
		},
		"status": {
			"example": "pending",
		},
	}
	resource, _ := AsStruct(x)

	//for {
	// Connect
	{
		op := &v1alpha1.Request{
			Op: &v1alpha1.Request_Connect{
				Connect: &v1alpha1.ConnectRequest{
					Resource: resource,
				},
			},
		}

		if err := stream.Send(op); err != nil {
			panic(err)
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("error receiving message: %v", err)
		}
		fmt.Println(resp)
	}
	// Disconnect
	{
		op := &v1alpha1.Request{
			Op: &v1alpha1.Request_Disconnect{
				Disconnect: &v1alpha1.DisconnectRequest{},
			},
		}

		if err := stream.Send(op); err != nil {
			panic(err)
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("error receiving message: %v", err)
		}
		fmt.Println(resp)
	}

	//}
}

// AsStruct gets the supplied struct from the supplied managed resource.
func AsStruct(mg any) (*structpb.Struct, error) {
	// We try to avoid a JSON round-trip if mg is backed by unstructured data.
	// Any type that is or embeds *unstructured.Unstructured has this method.
	if u, ok := mg.(interface{ UnstructuredContent() map[string]any }); ok {
		s, err := structpb.NewStruct(u.UnstructuredContent())
		return s, errors.Wrapf(err, "cannot create new Struct from %T", u)
	}

	b, err := json.Marshal(mg)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot marshal %T to JSON", mg)
	}
	s := &structpb.Struct{}
	return s, errors.Wrapf(protojson.Unmarshal(b, s), "cannot unmarshal JSON from %T into %T", mg, s)
}
