package sync

import (
	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap"
	synctypes "github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
)

func GetEmailFromInbox(mailClient imap.MailClient, banks []synctypes.BankDelegate) ([]synctypes.BankMessage, error) {
	const inboxMailbox = "INBOX"

	since := GetLastProcessedDate()

	var messages []synctypes.BankMessage

	for _, bank := range banks {
		msgs, err := mailClient.GetMessages(inboxMailbox, since, bank.FilterMessage)
		if err != nil {
			return nil, err
		}

		for _, msg := range msgs {
			bankMsg := synctypes.BankMessage{
				Message: msg,
				Bank:    bank,
			}

			messages = append(messages, bankMsg)
		}
	}

	return messages, nil
}

func ArchiveEmailsOfSuccessfulTransactions(mailClient imap.MailClient, successfulTransactions []*synctypes.TransactionInfo) {
	const archivedMailbox = "[Gmail]/All Mail"
	mailboxes, err := mailClient.GetMailBoxes()
	if err == nil {
		found := false
		for _, mailbox := range mailboxes {
			if mailbox == archivedMailbox {
				found = true
				break
			}
		}

		if !found {
			panic("archive mailbox not found " + archivedMailbox)
		}
	}

	var msgsIds []uint32
	for _, t := range successfulTransactions {
		msgsIds = append(msgsIds, t.MsgId)
	}
	err = mailClient.Move(msgsIds, archivedMailbox)
	if err != nil {
		panic(err)
	}
}
