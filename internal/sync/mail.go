package sync

import (
	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap"
	imaptypes "github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap/types"
	synctypes "github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
)

func GetEmailFromInbox(mailClient imap.MailClient, bank synctypes.BankDelegate) ([]imaptypes.Message, error) {
	const inboxMailbox = "INBOX"

	since := GetLastProcessedDate()
	messages, err := mailClient.GetMessages(inboxMailbox, since, bank.FilterMessage)
	if err != nil {
		return nil, err
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
