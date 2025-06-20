package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vrypan/farcaster-go/farcaster"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
)

// Used to add an extra HTTP header to requests.
// Required by Neynar
func apiKeyInterceptor(header, value string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md := metadata.Pairs(header, value)
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func main() {
	// set your hub addr:port here
	// port is usually 3383 for the gRPC endpoint but
	// in case of Neynar, it's 443
	hubAddr := "hub-grpc-api.neynar.com:443"

	// Is the server using SSL?
	useSsl := true
	cred := insecure.NewCredentials()
	if useSsl {
		cred = credentials.NewClientTLSFromCert(nil, "")
	}

	// Does the server expect a special API key in headers?
	// hubApiKey := "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	hubApiKey := ""
	var interceptor grpc.UnaryClientInterceptor
	if hubApiKey != "" {
		interceptor = apiKeyInterceptor("x-api-key", hubApiKey)
	}

	conn, err := grpc.Dial(
		hubAddr,
		grpc.WithTransportCredentials(cred),
		grpc.WithUnaryInterceptor(interceptor),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(20*1024*1024)),
	)

	defer conn.Close()

	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	client := farcaster.NewHubServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fetch the last 10 casts by fid 280
	var fid uint64 = 280
	var pageSize uint32 = 10
	var reverse bool = true

	res, err := client.GetCastsByFid(ctx, &farcaster.FidRequest{Fid: fid, Reverse: &reverse, PageSize: &pageSize})
	if err != nil {
		log.Fatalf("Failed to get casts by fid: %v", err)
	}

	// convert the result to json and print it
	jsonBytes, err := protojson.Marshal(res)
	fmt.Printf("%s\n", jsonBytes)
}
