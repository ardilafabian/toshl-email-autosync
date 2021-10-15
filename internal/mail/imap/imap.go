package imap

import (
	_imap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"log"
	"time"
)

type Mailbox string

type Message struct {
	_imap.Message
}

type Filter func(message Message) bool

type MailClient interface {
	GetMailBoxes() ([]Mailbox, error)
	GetMessages(mailbox Mailbox, since time.Time, filter Filter) ([]Message, error)
	Move(messages []Message, destMailbox Mailbox) error
	Logout() error
}

func GetMailClient(addr, username, password string) (MailClient, error) {
	emailClient, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}

	if err := emailClient.Login(username, password); err != nil {
		return nil, err
	}
	log.Println("Connected and logged in")

	return &mailClientImpl{client: emailClient}, nil
}

type mailClientImpl struct {
	client *client.Client
}

func (m mailClientImpl) GetMailBoxes() ([]Mailbox, error) {
	rawMailboxes := make(chan *_imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- m.client.List("", "*", rawMailboxes)
	}()

	var mailboxes []Mailbox
	for m := range rawMailboxes {
		mailbox := Mailbox(m.Name)
		mailboxes = append(mailboxes, mailbox)
	}

	return mailboxes, nil
}

func (m mailClientImpl) GetMessages(mailbox Mailbox, since time.Time, filter Filter) ([]Message, error) {
	panic("implement me")
}

func (m mailClientImpl) Move(messages []Message, destMailbox Mailbox) error {
	panic("implement me")
}

func (m mailClientImpl) Logout() error {
	if err := m.client.Logout(); err != nil {
		return err
	}

	return nil
}
