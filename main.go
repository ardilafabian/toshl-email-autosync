package main

import (
	"fmt"

	mail "github.com/Philanthropists/toshl-email-autosync/mail"
	gmail "github.com/Philanthropists/toshl-email-autosync/mail/gmail"
)

func main() {
	var service mail.Service

	service = &gmail.GmailService{}
	service.AuthenticateService()

	filters := []mail.Filter {
		"from:alertasynotificaciones@bancolombia.com.co",
		"after:2020/01/05",
	}

	for _, msg := range service.GetMessages(filters) {
		fmt.Println(msg)
	}
}