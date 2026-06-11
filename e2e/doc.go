// Package e2e contains end-to-end tests for the crank CLI.
//
// These tests are heavier than the unit/integration tests in
// internal/bootstrap: they build the real `crank` binary, exercise its
// command surface, and scaffold projects that are then compiled with the Go
// toolchain to prove the generated code is valid. Because they invoke `go get`
// for the generated projects, they require network access.
//
// They are guarded by the `e2e` build tag so they do not run during the
// ordinary `go test ./...` cycle. Run them explicitly with:
//
//	go test -tags e2e ./e2e/...
//
// or via the helper script:
//
//	./scripts/test.sh e2e
package e2e
