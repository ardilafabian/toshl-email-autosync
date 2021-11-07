package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"io"
	"log"
	"os"

	"github.com/Philanthropists/toshl-email-autosync/internal/market/investment-fund/bancolombia"
	"github.com/Philanthropists/toshl-email-autosync/internal/market/rapidapi"

	toshlclient "github.com/Philanthropists/toshl-go"
)

const credentialsFile = "credentials.json"

func GetStock() {
	api := rapidapi.RapidAPI{}
	err := api.GetCredentialsFromFile("rapidapi-keys.json")
	if err != nil {
		log.Fatal(err)
	}

	value, err := api.GetCurrentValue(rapidapi.USDCOP)
	if err != nil {
		log.Println("Error getting stock: ", err)
	}

	fmt.Printf("Current USD/COP value: %f\n", value)
}

func GetInvestmentFunds() {
	const fundName = "Renta Sostenible Global"
	list, err := bancolombia.GetAvailableInvestmentFundsBasicInfo()
	if err != nil {
		log.Println(err)
	}

	var fundId bancolombia.InvestmentFundId
	found := false
	for _, fundInfo := range list {
		if fundInfo.Name == fundName {
			fundId = fundInfo.Nit
			found = true
			break
		}
	}

	if !found {
		log.Printf("fund name [%s] not found in list", fundName)
		return
	}

	fund, err := bancolombia.GetInvestmentFundById(fundId)
	if err != nil {
		log.Println(err)
	}

	log.Printf("%+v", fund)
}

func GetToshlAccounts(auth types.Auth) {
	client := toshlclient.NewClient(auth.ToshlToken, nil)

	accounts, err := client.Accounts(nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", accounts)
}

type Options struct {
	DryRun bool
	Debug  bool
}

func getOptions() Options {
	defer flag.Parse()

	var options Options

	flag.BoolVar(&options.DryRun, "dryRun", false, "Tell what will happen but not execute")
	flag.BoolVar(&options.Debug, "debug", false, "Output debug output")

	return options
}

func getAuth() (types.Auth, error) {
	credFile, err := os.Open(credentialsFile)
	if err != nil {
		return types.Auth{}, err
	}
	defer credFile.Close()

	authBytes, err := io.ReadAll(credFile)
	if err != nil {
		return types.Auth{}, err
	}

	var auth types.Auth
	err = json.Unmarshal(authBytes, &auth)
	if err != nil {
		return types.Auth{}, err
	}

	return auth, nil
}

func main() {
	_ = getOptions()

	auth, err := getAuth()
	if err != nil {
		panic(err)
	}

	sync.Run(context.Background(), auth)
}
