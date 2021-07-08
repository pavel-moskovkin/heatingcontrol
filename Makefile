.PHONY: bin
bin:
	mkdir -p bin/

.PHONY: build
build: bin
	go build -o bin/heatingcontrol ./cmd/*.go

.PHONY: vendor
vendor:
	rm -rf vendor
	go mod vendor

.PHONY: test
test:
	go test ./... -v
