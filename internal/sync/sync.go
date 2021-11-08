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

const notificationFormat = `Synced transactions:
	%d successful
	%d failures
	%d failed to parse`

type txsStatus struct {
	SuccessfulTxs []*types.TransactionInfo
	FailedTxs     []*types.TransactionInfo
	ParseFailures int64
}

func Run(ctx context.Context, auth types.Auth) error {
	var status txsStatus

	log := logger.GetLogger()
	defer log.Sync()

	defer func() {
		log.Infow("Synced transactions",
			"successful", len(status.SuccessfulTxs),
			"failed", len(status.FailedTxs),
			"failed_to_parse", status.ParseFailures,
		)

		shouldNotify := status.ParseFailures > 0
		shouldNotify = shouldNotify || len(status.FailedTxs) > 0
		shouldNotify = shouldNotify || len(status.SuccessfulTxs) > 0

		if shouldNotify && auth.TwilioAccountSid != "" {
			msg := fmt.Sprintf(notificationFormat, len(status.SuccessfulTxs), len(status.FailedTxs), status.ParseFailures)
			SendNotifications(auth, msg)
		}
	}()

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

	var transactions []*types.TransactionInfo
	transactions, status.ParseFailures = ExtractTransactionInfoFromMessages(msgs)

	if status.ParseFailures > 0 {
		log.Infow("Had failures extracting information from messages",
			"failures", status.ParseFailures,
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

	status.SuccessfulTxs, status.FailedTxs = CreateEntries(toshlClient, transactions, mappableAccounts, internalCategoryId)

	ArchiveEmailsOfSuccessfulTransactions(mailClient, status.SuccessfulTxs)

	if err := UpdateLastProcessedDate(status.FailedTxs); err != nil {
		return fmt.Errorf("failed to update last processed date: %s", err)
	}

	return nil
}
