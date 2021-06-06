package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getAuthenticatedMailService() *gmail.Service {
	ctx := context.Background()
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	return srv
}

func processAndGatherMessages(srv *gmail.Service, in <-chan string, out chan<- *gmail.Message) {
	const concurrentJobs = 10
	const user = "me"

	var wg sync.WaitGroup
	wg.Add(concurrentJobs)
	for i := 0; i < concurrentJobs; i++ {
		go func() {
			for msgId := range in {
				msg, _ := srv.Users.Messages.Get(user, msgId).Do()
				out <- msg
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(out)
	}()
}

func getMessageList(srv *gmail.Service, msgIdCh chan<- string) {
	const user = "me"
	filters := "from:alertasynotificaciones@bancolombia.com.co after:2020/01/05"

	finished := false
	var msgs *gmail.ListMessagesResponse
	var err error
	var nextPageToken string
	for !finished {
		msgs, err = srv.Users.Messages.List(user).Q(filters).PageToken(nextPageToken).Do()
		if err != nil {
			log.Fatal(err)
		}

		for _, msg := range msgs.Messages {
			msgIdCh <- msg.Id
		}

		nextPageToken = msgs.NextPageToken
		if nextPageToken == "" {
			finished = true
		}
	}
	close(msgIdCh)
}

func getMessages(srv *gmail.Service) {
	in := make(chan string)
	out := make(chan *gmail.Message)

	go getMessageList(srv, in)
	processAndGatherMessages(srv, in, out)

	for msg := range out {
		// fmt.Println(msg)
		if msg != nil {
			fmt.Printf("id: %s, body: %s\n", msg.Id, msg.Snippet)
		}
	}
}

func main() {
	srv := getAuthenticatedMailService()
	getMessages(srv)
}
