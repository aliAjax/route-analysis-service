.PHONY: fmt vet test run build
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')
vet:
	go vet ./...
test:
	go test ./...
run:
	go run ./cmd/route-service
build:
	go build ./cmd/route-service
