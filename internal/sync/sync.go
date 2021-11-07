package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/bank"
	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap"
	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
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

func ExtractTransactionInfoFromMessages(msgs []types.BankMessage) ([]*types.TransactionInfo, int64) {
	log := logger.GetLogger()
	var failures int64

	var transactions []*types.TransactionInfo
	for _, bankMsg := range msgs {
		t, err := bankMsg.Bank.ExtractTransactionInfoFromMessage(bankMsg.Message)
		if err == nil {
			transactions = append(transactions, t)
		} else {
			log.Errorw("Error processing message",
				"error", err,
				"msgId", bankMsg.SeqNum,
			)
			failures++
		}
	}

	return transactions, failures
}

func getEarliestDateFromTxs(txs []*types.TransactionInfo) time.Time {
	earliestDate := time.Now().Add(-24 * time.Hour)
	for _, tx := range txs {
		date := tx.Date
		if date.Before(earliestDate) {
			earliestDate = date
		}
	}

	return earliestDate
}

func Run(ctx context.Context, auth types.Auth) error {
	log := logger.GetLogger()
	defer log.Sync()

	banks := bank.GetBanks()

	mailClient, err := imap.GetMailClient(auth.Addr, auth.Username, auth.Password)
	if err != nil {
		return err
	}
	defer mailClient.Logout()

	msgs, err := GetEmailFromInbox(mailClient, banks)
	if err != nil {
		return err
	}

	transactions, failures := ExtractTransactionInfoFromMessages(msgs)

	if failures > 0 {
		log.Infow("Had failures extracting information from messages",
			"failures", failures,
		)
	}

	if len(transactions) == 0 {
		log.Info("no transactions to process, exiting ... ")
		return nil
	}

	log.Debug("Transactions to process")
	for i, t := range transactions {
		log.Debugf("%d: %+v", i, t)
	}

	toshlClient := toshl.NewApiClient(auth.ToshlToken)
	internalCategoryId := CreateInternalCategoryIfAbsent(toshlClient)

	accounts, err := toshlClient.GetAccounts()
	if err != nil {
		return err
	}

	log.Debug("Account")
	for i, a := range accounts {
		log.Debugf("%d: %s", i, a.Name)
	}

	mappableAccounts := GetMappableAccounts(accounts)

	log.Debug("Mappable accounts")
	for name, account := range mappableAccounts {
		log.Debugf("%s: %s", name, account.Name)
	}

	successfulTxs, failedTxs := CreateEntries(toshlClient, transactions, mappableAccounts, internalCategoryId)

	ArchiveEmailsOfSuccessfulTransactions(mailClient, successfulTxs)

	log.Infow("Synced transactions",
		"successful", len(successfulTxs),
		"failed", len(failedTxs),
	)
	if len(successfulTxs) > 0 && auth.TwilioAccountSid != "" {
		msg := fmt.Sprintf("Synced transactions: %d successful- %d failed", len(successfulTxs), len(failedTxs))
		SendNotifications(auth, msg)
	}

	if err := UpdateLastProcessedDate(failedTxs); err != nil {
		return fmt.Errorf("failed to update last processed date: %s", err)
	}

	return nil
}
