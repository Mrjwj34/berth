## Summary

<!-- Briefly describe what this change does and why it is needed. Conventional commit type belongs in the PR title (`feat:`, `fix:`, `refactor:`, etc.). -->

## Type of Change

- [ ] `feat`: A new feature
- [ ] `fix`: A bug fix
- [ ] `docs`: Documentation updates
- [ ] `refactor`: Code refactoring without behavioral changes
- [ ] `test`: Adding or updating tests
- [ ] `ci` / `chore`: Maintenance, CI, or tooling changes

## Test Plan

- [ ] `gofmt -l cmd internal` is clean
- [ ] `go vet ./cmd/... ./internal/...`
- [ ] `go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./cmd/... ./internal/...`
- [ ] `go test -race ./cmd/... ./internal/...`
- [ ] Manual verification / CLI test commands executed:
