FROM golang:1.18.0 as builder

ARG COMMIT=dev

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

ENV GOOS=linux GOARCH=amd64 CGO_ENABLED=0
ENV LOC=/usr/local/bin

COPY . .
RUN GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=${CGO_ENABLED} \
		 go build \
		 -ldflags="-s -w -X main.GitCommit=${COMMIT}" \
		 -gcflags=-G=3 \
		 -o ${LOC}/main cmd/aws-lambda/main.go

FROM alpine:3.15.0

WORKDIR /

# Needed for getting America/Bogota Location
RUN apk add --no-cache tzdata=2022a-r0

COPY credentials.json .
COPY --from=builder /usr/local/bin/main ./main

ENTRYPOINT [ "/main"]
