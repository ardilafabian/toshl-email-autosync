package imap

import (
	"errors"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

var (
	// DefaultHTTPGetAddress Default Address
	DefaultHTTPGetAddress = "https://checkip.amazonaws.com"

	// ErrNoIP No IP found in response
	ErrNoIP = errors.New("no IP in HTTP response")

	// ErrNon200Response non 200 status code in response
	ErrNon200Response = errors.New("non 200 Response found")

	localLocation *time.Location
)

func init() {
	var err error
	localLocation, err = time.LoadLocation("America/Bogota")
	if err != nil {
		panic(err)
	}
}

func TestImpl(addr string, username string, password string) {
	//actualTime := time.Now()

	//(events.APIGatewayProxyResponse, error)

	log.Println("Connecting to server...")

	// Connect to server
	emailClient, err := client.DialTLS(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	// Don't forget to logout
	defer emailClient.Logout()

	// Login
	if err := emailClient.Login(username, password); err != nil {
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

	//imap.ParseFlagsOp()

	criteria := imap.NewSearchCriteria()
	criteria.Since = time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC)
	ids, err := emailClient.Search(criteria)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("IDs found:", len(ids))
	if len(ids) == 0 {
		log.Println("Nothing found")
		return
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *imap.Message, 100)
	done = make(chan error, 1)

	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}
	go func() {
		emailClient.Fetch(seqset, items, messages)
	}()

	for msg := range messages {
		shouldSkip := true
		if msg != nil && msg.Envelope != nil {
			for _, address := range msg.Envelope.From {
				from := address.Address()
				if from == "alertasynotificaciones@notificacionesbancolombia.com" {
					shouldSkip = false
				}
			}
		}

		if shouldSkip {
			continue
		}

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
				text := string(b)
				log.Printf("Email length: %v\n", len(text))

				shouldProcess := strings.Contains(text, "Pago")
				shouldProcess = shouldProcess || strings.Contains(text, "Compra")
				shouldProcess = shouldProcess || strings.Contains(text, "Transferencia")

				if shouldProcess {
					log.Printf("%s\n", msg.Envelope.Subject)
					log.Printf("%s\n", msg.Envelope.Date.In(localLocation))
				}
				// log.Printf("Email [%v]: %v\n", msg.Envelope.Subject, len(string(b)))
			}
		}
	}

	log.Println("Done!")

	//return events.APIGatewayProxyResponse{
	//	Body:       "",
	//	StatusCode: 200,
	//}, nil
}
