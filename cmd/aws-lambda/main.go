package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	concurrency "sync"

	"github.com/Philanthropists/toshl-email-autosync/internal/market"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/common"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"github.com/aws/aws-lambda-go/lambda"
)

const credentialsFile = "credentials.json"

var GitCommit string

func getAuth(rawAuth []byte) (types.Auth, error) {
	auth := types.Auth{}

	err := json.Unmarshal(rawAuth, &auth)
	if err != nil {
		return types.Auth{}, err
	}

	return auth, nil
}

func HandleRequest(ctx context.Context) error {
	common.PrintVersion(GitCommit)

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

	var wg concurrency.WaitGroup
	wg.Add(2)

	go func() {
		errThis := sync.Run(ctx, auth)
		if errThis != nil {
			err = errThis
		}
		wg.Done()
	}()

	go func() {
		errThis := market.Run(ctx, auth)
		if errThis != nil {
			err = errThis
		}
		wg.Done()
	}()

	wg.Wait()

	return err
}

func main() {
	lambda.Start(HandleRequest)
}
