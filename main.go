package main

import (
	"fmt"
	"github.com/Philanthropists/toshl-email-autosync/mail"
	"github.com/Philanthropists/toshl-email-autosync/mail/gmail"
	"log"
	"sync"

	"github.com/Philanthropists/toshl-email-autosync/market/investment_fund/bancolombia"
	"github.com/Philanthropists/toshl-email-autosync/market/rapidapi"
	toshl "github.com/Philanthropists/toshl-go"
)

func getMail() {
	var service mail.Service = &gmail.GmailService{}
	service.AuthenticateService()

	filters := []mail.Filter{
		{
			Type:  mail.FromFilter,
			Value: "alertasynotificaciones@bancolombia.com.co",
		},
		{
			Type:  mail.AfterFilter,
			Value: "2020/01/05",
		},
	}

	for _, msg := range service.GetMessages(filters) {
		fmt.Println(msg)
	}
}

func getStock() {
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

func getInvestmentFunds() {
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

func getToshlInfo() {
	token := GetDefaultToshlToken()
	client := toshl.NewClient(token, nil)

	accounts, err := client.Accounts(nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", accounts)
}

func waitToFinish(wg *sync.WaitGroup, f func()) {
	f()
	wg.Done()
}

func main() {
	var wg sync.WaitGroup
	wg.Add(4)

	go waitToFinish(&wg, getMail)
	go waitToFinish(&wg, getStock)
	go waitToFinish(&wg, getInvestmentFunds)
	go waitToFinish(&wg, getToshlInfo)

	wg.Wait()
}
