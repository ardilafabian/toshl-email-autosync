package main

import (
	"context"
	"encoding/json"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync"
	"github.com/aws/aws-lambda-go/lambda"
	"io"
	"os"
)

const credentialsFile = "credentials.json"

func getAuth(rawAuth []byte) (sync.Auth, error) {
	auth := sync.Auth{}

	err := json.Unmarshal(rawAuth, &auth)
	if err != nil {
		return sync.Auth{}, err
	}

	return auth, nil
}

func HandleRequest(ctx context.Context) error {
	credFile, err := os.Open(credentialsFile)
	if err != nil {
		return err
	}
	defer credFile.Close()

	authBytes, err := io.ReadAll(credFile)
	if err != nil {
		return err
	}

	auth, err := getAuth(authBytes)
	if err != nil {
		return err
	}

	return sync.Run(ctx, auth)
}

func main() {
	lambda.Start(HandleRequest)
}
