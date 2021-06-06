package main

import (
	"errors"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"io"
	"io/ioutil"
	"log"
)

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("No IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("Non 200 Response found")
)

func handler() {
	//actualTime := time.Now()

	//(events.APIGatewayProxyResponse, error)

	log.Println("Connecting to server...")

	// Connect to server
	emailClient, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Don't forget to logout
	defer emailClient.Logout()

	// Login
	if err := emailClient.Login("<username>", "<password>"); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	// List mailboxes
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- emailClient.List("", "*", mailboxes)
	}()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println(m.Name)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	// Select INBOX
	mbox, err := emailClient.Select("INBOX", true)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Flags for INBOX:", mbox.Flags)

	// Get the last 4 messages
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > 100 {
		// We're using unsigned integers here, only subtract if the result is > 0
		from = mbox.Messages - 100
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)

	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}
	go func() {
		emailClient.Fetch(seqset, items, messages)
	}()

	for msg := range messages {
		t := msg.GetBody(&section)
		mr, _ := mail.CreateReader(t)
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}

			switch p.Header.(type) {
			case *mail.InlineHeader:
				// This is the message's text (can be plain-text or HTML)
				b, _ := ioutil.ReadAll(p.Body)
				log.Println("Got text: %v", len(string(b)))
			}
		}
	}

	log.Println("Done!")

	//return events.APIGatewayProxyResponse{
	//	Body:       "",
	//	StatusCode: 200,
	//}, nil
}

func main() {
	//lambda.Start(handler)
	handler()
}
