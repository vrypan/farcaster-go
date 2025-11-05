SNAPCHAIN_VER := $(shell cat SNAPCHAIN_VERSION 2>/dev/null || echo "unset")

PROTO_FILES = $(wildcard proto/*.proto)

# Colors for output
GREEN = \033[0;32m
NC = \033[0m

all: proto-bindings

# Compile .proto files, touch stamp file
proto-bindings: fetch-proto proto/.downloaded_version
	@echo -e "$(GREEN)Compiling .proto files...$(NC)"
	@command -v protoc >/dev/null || (echo "Error: protoc not found in PATH" && exit 1)
	protoc --proto_path=proto --go_out=. --go-grpc_out=. \
        $(foreach f,$(PROTO_FILES),--go_opt=M$(notdir $(f))=farcaster/ --go-grpc_opt=M$(notdir $(f))=farcaster/) \
        $(PROTO_FILES)
	@echo "package farcaster" > farcaster/doc.go
	@touch $@

fetch-proto: SNAPCHAIN_VERSION proto/.downloaded_version
	@echo -e "$(GREEN)Downloading proto files (Hubble v$(SNAPCHAIN_VER))...$(NC)"
	curl -s -L "https://codeload.github.com/farcasterxyz/snapchain/tar.gz/refs/tags/v$(SNAPCHAIN_VER)" \
	    | tar -zxvf - -C . --strip-components 2 "snapchain-$(SNAPCHAIN_VER)/src/proto/"
	@echo -e "$(GREEN)Patching proto files to avoid namespace conflictsi with libp2p...$(NC)"
	@mv proto/message.proto proto/snapmsg_fc.proto
	@sed -i '' 's/import "message.proto"/import "snapmsg_fc.proto"/g' proto/gossip.proto proto/rpc.proto proto/hub_event.proto proto/replication.proto proto/blocks.proto proto/request_response.proto
	@for f in proto/*.proto; do \
		awk 'NR==1{print; print "package farcaster;"; print ""; next}1' "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; \
	done
	@echo "$(SNAPCHAIN_VER)" > proto/.downloaded_version
	@touch $@

proto/.downloaded_version:
	@touch $@

clean:
	@echo -e "$(GREEN)Deleting protobuf definitions...$(NC)"
	rm -f proto/* fetch-proto
	@echo -e "$(GREEN)Deleting protobuf bindings...$(NC)"
	rm -f farcaster/*.pb.go farcaster/*.pb.gw.go
	rm -f proto-bindings

.PHONY: all clean
