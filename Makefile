.PHONY: mod
mod:
	go mod download

.PHONY: test
test:
	go test ./... -v
