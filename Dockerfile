FROM golang:1.17.8 as builder

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
RUN apk add --no-cache tzdata

ADD https://github.com/aws/aws-lambda-runtime-interface-emulator/releases/latest/download/aws-lambda-rie /usr/bin/aws-lambda-rie
COPY entry.sh .
RUN chmod 755 /usr/bin/aws-lambda-rie && chmod 755 entry.sh

COPY credentials.json .
COPY --from=builder /usr/local/bin/main ./main

ENTRYPOINT [ "/entry.sh", "/main" ]
