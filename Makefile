.PHONY: build
build: bin vendor fmt
	go build -o bin cmd/run/run.go
	go build -o bin cmd/aws-lambda/main.go
	cp credentials.json bin/

build-for-lambda: bin clean vendor fmt
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/main cmd/aws-lambda/main.go
	cp credentials.json bin/

bin:
	mkdir -p bin

.PHONY: fmt
fmt: staticcheck
	go fmt ./...

.PHONY: staticcheck
staticcheck: vet
	go install honnef.co/go/tools/cmd/staticcheck@latest
	$(GOPATH)/bin/staticcheck ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: vendor
vendor: tidy
	go mod vendor

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin/*
