# Git Workflow for sb-messenger-sdk

This repository contains the Slidebolt Messenger SDK, providing the interfaces and NATS-based implementation for communication between services. It has zero internal Slidebolt dependencies.

## Dependencies
- **Internal:** None. This is a base repository.
- **External:** 
  - `github.com/nats-io/nats.go`: Official NATS client.
  - `github.com/nats-io/nats-server/v2`: NATS server for testing.

## Build Process
- **Type:** Pure Go Library (Shared Module).
- **Consumption:** Imported as a module dependency in other Go projects via `go.mod`.
- **Artifacts:** No standalone binary or executable is produced.
- **Validation:** 
  - Validated through unit tests: `go test -v ./...`
  - Validated by its consumers during their respective build/test cycles.

## Pre-requisites & Publishing
As a base SDK, `sb-messenger-sdk` must be updated and published before any repositories that rely on updated messaging patterns.

**Before publishing:**
1. Determine current tag: `git tag | sort -V | tail -n 1`
2. Ensure all local tests pass: `go test -v ./...`
3. Check impact on major consumers like `sb-api`, `sb-manager`, and all `plugin-*` repositories.

**Publishing Order:**
1. Update `sb-messenger-sdk`.
2. Determine next semantic version (e.g., `v1.0.4`).
3. Commit and push the changes to `main`.
4. Tag the repository: `git tag v1.0.4`.
5. Push the tag: `git push origin v1.0.4`.
6. Update dependent repositories using `go get github.com/slidebolt/sb-messenger-sdk@v1.0.4`.

## Update Workflow & Verification
1. **Modify:** Update messaging logic or interfaces in `messenger.go` or `commands.go`.
2. **Verify Local:**
   - Run `go mod tidy`.
   - Run `go test ./...`.
3. **Commit:** Ensure the commit message clearly describes the SDK change.
4. **Tag & Push:** (Follow the Publishing Order above).
