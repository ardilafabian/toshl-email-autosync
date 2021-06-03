.PHONY: build

fmt:
	go fmt .

build: fmt
	go build -o .

clean:
	git clean -xdf
