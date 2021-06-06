.PHONY: build

init:
	mkdir -p bin

vet:
	go vet ./...

staticcheck: vet
	go install honnef.co/go/tools/cmd/staticcheck@latest
	$(GOPATH)/bin/staticcheck ./...

fmt: staticcheck
	go fmt ./...

tidy:
	go mod tidy

vendor: tidy
	go mod vendor

build: init vendor fmt
	go build -o bin

clean:
	rm -f bin/*
