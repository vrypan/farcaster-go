# libp2p Compatibility Patches

This document explains the automatic protobuf patching applied during the build process to ensure compatibility with libp2p.

## The Problem

When using `farcaster-go` together with `go-libp2p` in the same project, a protobuf namespace conflict occurs:

```
panic: proto: file "message.proto" has a name conflict over Message
    previously from: "github.com/libp2p/go-libp2p/p2p/transport/webrtc/pb"
    currently from:  "github.com/vrypan/farcaster-go/farcaster"
```

This happens because:

1. **Farcaster's protocol** defines a `Message` type in `message.proto`
2. **libp2p's webrtc transport** also defines a `Message` type in `message.proto`
3. Both proto files register globally with the same filename and type name
4. Go's protobuf runtime detects this conflict and panics

This is a **global namespace collision** in the protobuf registry, not a Go package collision.

## The Solution

We automatically patch the downloaded proto files to avoid the conflict by:

1. **Renaming the file**: `message.proto` → `snapmsg_fc.proto`
2. **Adding package declarations**: Add `package farcaster;` to all proto files
3. **Updating imports**: Update all references from `message.proto` to `snapmsg_fc.proto`

These patches are applied automatically by the Makefile every time proto files are downloaded.

## What Gets Changed

### File Renaming

**Before:**
```
proto/message.proto
```

**After:**
```
proto/snapmsg_fc.proto
```

### Package Declaration

**Before:**
```protobuf
syntax = "proto3";

import "username_proof.proto";

message Message {
  ...
}
```

**After:**
```protobuf
syntax = "proto3";
package farcaster;

import "username_proof.proto";

message Message {
  ...
}
```

This is applied to **all 12 proto files** to ensure consistent namespacing.

### Import Updates

Files that import `message.proto` are updated:
- `gossip.proto`
- `rpc.proto`
- `hub_event.proto`
- `replication.proto`
- `blocks.proto`
- `request_response.proto`

**Before:**
```protobuf
import "message.proto";
```

**After:**
```protobuf
import "snapmsg_fc.proto";
```

## Impact on Users

### ✅ No Breaking Changes for Standard Usage

**Go API remains identical:**
```go
import pb "github.com/vrypan/farcaster-go/farcaster"

// All of this works exactly the same
msg := &pb.Message{...}
data, _ := proto.Marshal(msg)
proto.Unmarshal(data, &msg)
```

**Wire format is unchanged:**
- Messages encoded before/after the patch are 100% compatible
- Network communication works between patched and unpatched versions
- All field names, numbers, and types remain identical

**JSON output is unchanged (for basic usage):**
```go
jsonBytes, _ := protojson.Marshal(msg)
// Output is identical
```

### ⚠️ Potential Impact (Advanced Usage)

**1. Protobuf `Any` types:**

Type URLs now include the package name:

**Before:**
```json
{"@type": "type.googleapis.com/Message"}
```

**After:**
```json
{"@type": "type.googleapis.com/farcaster.Message"}
```

**2. Protobuf reflection by name:**

```go
// Must now use full name
registry.FindMessageByName("farcaster.Message")  // not "Message"
msg.ProtoReflect().Descriptor().FullName()       // returns "farcaster.Message"
```

**3. Custom type resolvers:**

If you have custom protobuf type resolution logic, update it to use the full type name `farcaster.Message`.

## How It Works

The patching is implemented in the `Makefile` in the `fetch-proto` target:

```makefile
fetch-proto: SNAPCHAIN_VERSION proto/.downloaded_version
	@echo -e "$(GREEN)Downloading proto files (Hubble v$(SNAPCHAIN_VER))...$(NC)"
	curl -s -L "https://codeload.github.com/farcasterxyz/snapchain/tar.gz/refs/tags/v$(SNAPCHAIN_VER)" \
	    | tar -zxvf - -C . --strip-components 2 "snapchain-$(SNAPCHAIN_VER)/src/proto/"
	@echo -e "$(GREEN)Patching proto files to avoid namespace conflicts with libp2p...$(NC)"
	@mv proto/message.proto proto/snapmsg_fc.proto
	@sed -i '' 's/import "message.proto"/import "snapmsg_fc.proto"/g' \
	    proto/gossip.proto proto/rpc.proto proto/hub_event.proto \
	    proto/replication.proto proto/blocks.proto proto/request_response.proto
	@for f in proto/*.proto; do \
		awk 'NR==1{print; print "package farcaster;"; print ""; next}1' "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; \
	done
	@echo "$(SNAPCHAIN_VER)" > proto/.downloaded_version
	@touch $@
```

**Automatic application:**
- Every `make clean && make` re-downloads proto files and applies patches
- No manual intervention needed
- Patches are always current with upstream protocol definitions

## Compatibility with Upstream

**Tracking Farcaster protocol:**
- We download proto files directly from the official Farcaster Snapchain repository
- Only naming/namespacing is changed, not the actual protocol
- Updates to the protocol are automatically picked up via version bumps

**Version tracking:**
- The `SNAPCHAIN_VERSION` file specifies which upstream version to use
- Currently: v0.10.0

## Migration Guide

### For Existing Projects

If you're updating from an unpatched version of `farcaster-go`:

**Most projects:** No changes needed. Just update the dependency.

**If you use `Any` types:** Update type URL handling to expect `farcaster.Message` instead of `Message`.

**If you use reflection:** Update code to use full type names (`farcaster.Message`).

### For New Projects

Just use this version - the patches are transparent for standard protobuf usage.

## Why Not Fix Upstream?

**Option 1: Change Farcaster's proto files**
Not possible.

**Option 2: Change libp2p's proto files**
Not possible.

**Option 3: Patch locally (our approach)**
- ✅ Maintains wire compatibility with Farcaster
- ✅ Maintains API compatibility with existing code
- ✅ Fixes the namespace conflict
- ✅ Automated and maintainable

## Technical Details

### Why Both Filename and Package?

**Filename change alone** solves the file registry conflict but not the type registry conflict (both define `Message`).

**Package alone** would solve the type conflict but requires changing all proto files to maintain consistency.

**Both together** ensures:
- No file registry conflicts (different filenames)
- No type registry conflicts (different packages)
- All Farcaster types are in the same namespace (`farcaster.*`)

### Registry Mechanics

Protobuf uses two global registries:

1. **File registry**: Maps `filename.proto` → `FileDescriptor`
2. **Type registry**: Maps `package.TypeName` → `MessageDescriptor`

Without a package, types register in the global namespace. With `package farcaster;`, types register as `farcaster.TypeName`.

## Testing

To verify the patches work:

```bash
# Clean rebuild
make clean
make

# Check package declarations were added
head -5 proto/snapmsg_fc.proto
# Should show: package farcaster;

# Check imports were updated
grep 'import.*snapmsg_fc' proto/*.proto
# Should show 6 files

# Test with libp2p
cd ../snapchain-listener-go
go build
./snapchain-listener
# Should connect without protobuf panics
```

## Questions?

If you encounter issues with these patches, please open an issue describing:
1. Your use case (standard protobuf, `Any` types, reflection, etc.)
2. The error message or unexpected behavior
3. Whether you're using libp2p in the same project

## References

- [Protobuf Namespace Conflicts](https://protobuf.dev/reference/go/faq#namespace-conflict)
- [Farcaster Protocol](https://github.com/farcasterxyz/snapchain)
- [libp2p](https://github.com/libp2p/go-libp2p)
