package sync

import (
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/twilio"
	"log"
)

func SendNotifications(auth types.Auth, msg string) {
	accountSid := auth.TwilioAccountSid
	authToken := auth.TwilioAuthToken
	fromNumber := auth.TwilioFromNumber
	toNumber := auth.TwilioToNumber

	client, err := twilio.NewClient(accountSid, authToken)
	if err != nil {
		log.Printf("error: could not instantiate twilio client: %s", err)
		return
	}

	_, err = client.SendSms(fromNumber, toNumber, msg)
	if err != nil {
		log.Printf("error: an error ocurred when sending notification sms")
		return
	}
}
