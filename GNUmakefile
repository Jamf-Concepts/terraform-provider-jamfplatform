default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e main.go internal/ tools/

fix:
	go fix ./...

test:
	go test -v -cover -count=1 -timeout=120s -p=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -count=1 -tags acceptance -timeout 120m -p=1 ./...

.PHONY: fmt fix lint test testacc build install generate