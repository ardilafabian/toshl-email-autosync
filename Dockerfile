FROM golang:1.17.7 as builder

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

FROM public.ecr.aws/lambda/provided:al2

COPY --from=builder /usr/local/bin/main /main
COPY credentials.json .

ENTRYPOINT [ "/main" ]
