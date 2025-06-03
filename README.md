# farcaster-go
Go bindings for farcaster protobufs.

# Usage

Get the package:
```
go get github.com/vrypan/farcaster-go/farcaster
```

```golang
// main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/vrypan/farcaster-go/farcaster"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	hubAddr := "some_server_dot_com:3383" // set your hub hostname here
	useSsl := false
	cred := insecure.NewCredentials()
	if useSsl {
		cred = credentials.NewClientTLSFromCert(nil, "")
	}
	conn, err := grpc.Dial(
		hubAddr,
		grpc.WithTransportCredentials(cred),
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
```

More examples will be added in [examples](./examples)
