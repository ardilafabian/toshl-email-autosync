package sync

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/bank/bancolombia"
	synctypes "github.com/Philanthropists/toshl-email-autosync/internal/sync/types"

	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap"
	imaptypes "github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/toshl"
)

var localLocation *time.Location

func init() {
	var err error
	localLocation, err = time.LoadLocation("America/Bogota")
	if err != nil {
		panic(err)
	}
}

func ExtractTransactionInfoFromMessages(bank synctypes.BankDelegate, msgs []imaptypes.Message) ([]*synctypes.TransactionInfo, int64) {
	var failures int64

	var transactions []*synctypes.TransactionInfo
	for _, msg := range msgs {
		t, err := bank.ExtractTransactionInfoFromMessage(msg)
		if err == nil {
			transactions = append(transactions, t)
		} else {
			log.Printf("Error processing message: %s", err)
			failures++
		}
	}

	return transactions, failures
}

func getEarliestDateFromTxs(txs []*synctypes.TransactionInfo) time.Time {
	earliestDate := time.Now().Add(-24 * time.Hour)
	for _, tx := range txs {
		date := tx.Date
		if date.Before(earliestDate) {
			earliestDate = date
		}
	}

	return earliestDate
}

func Run(ctx context.Context, auth synctypes.Auth) error {
	bank := bancolombia.Bancolombia{}

	mailClient, err := imap.GetMailClient(auth.Addr, auth.Username, auth.Password)
	if err != nil {
		return err
	}
	defer mailClient.Logout()

	msgs, err := GetEmailFromInbox(mailClient, bank)
	if err != nil {
		return err
	}

	transactions, failures := ExtractTransactionInfoFromMessages(bank, msgs)

	if failures > 0 {
		log.Printf("Had %d failures on extracting information from messages", failures)
	}

	if len(transactions) == 0 {
		log.Printf("no transactions to process, exiting ... ")
		return nil
	}

	for _, t := range transactions {
		log.Printf("%+v", t)
	}

	toshlClient := toshl.NewApiClient(auth.ToshlToken)
	internalCategoryId := CreateInternalCategoryIfAbsent(toshlClient)

	accounts, err := toshlClient.GetAccounts()
	if err != nil {
		return err
	}

	for _, a := range accounts {
		log.Printf("Accounts %+v", a)
	}

	mappableAccounts := GetMappableAccounts(accounts)

	for name, account := range mappableAccounts {
		log.Printf("Mappable accounts: [%s] : %+v", name, account)
	}

	successfulTxs, failedTxs := CreateEntries(toshlClient, transactions, mappableAccounts, internalCategoryId)

	ArchiveEmailsOfSuccessfulTransactions(mailClient, successfulTxs)

	msg := fmt.Sprintf("Synced transactions: %d sucessful - %d failed", len(successfulTxs), len(failedTxs))
	log.Println(msg)
	if len(successfulTxs) > 0 && auth.TwilioAccountSid != "" {
		SendNotifications(auth, msg)
	}

	if err := UpdateLastProcessedDate(failedTxs); err != nil {
		return fmt.Errorf("failed to update last processed date: %s", err)
	}

	return nil
}
