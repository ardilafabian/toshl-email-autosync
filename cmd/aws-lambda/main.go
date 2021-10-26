package main

import (
	"context"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync"
	"github.com/aws/aws-lambda-go/lambda"
)

func HandleRequest(ctx context.Context) error {
	return sync.Run(ctx)
}

func main() {
	lambda.Start(HandleRequest)
}
