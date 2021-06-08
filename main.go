package main

import (
	"fmt"
	"github.com/Philanthropists/toshl-email-autosync/mail"
	"github.com/Philanthropists/toshl-email-autosync/mail/gmail"
	"log"

	"github.com/Philanthropists/toshl-email-autosync/market/rapidapi"
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

	fmt.Printf("Current USD/COP value: %f", value)
}

func main() {
	getMail()
	getStock()
}
