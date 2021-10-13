package gmail

import (
	"context"
	"crypto/x509"
	//"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/Philanthropists/toshl-email-autosync/internal/mail/types"
	"github.com/golang-jwt/jwt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	const tokFile = "token.json"
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

func GetGmailService() types.Service {
	return &gmailService{}
}

type gmailService struct {
	srv *gmail.Service
}

type ServiceAccountPK struct {
	Type                    string `json:"type"`
	ProjectId               string `json:"project_id"`
	PrivateKeyId            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientId                string `json:"client_id"`
	AuthUri                 string `json:"auth_uri"`
	TokenUri                string `json:"token_uri"`
	AuthProviderX509CertUrl string `json:"auth_provider_x509_cert_url"`
	ClientX509CertUrl       string `json:"client_x509_cert_url"`
}

func GetCredentialsFromServiceAccount() {
	credentials_file, err := ioutil.ReadFile("service-account-pk.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	var sv ServiceAccountPK
	err = json.Unmarshal(credentials_file, &sv)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", sv.PrivateKey)
	block, _ := pem.Decode([]byte(sv.PrivateKey))
	if block == nil || block.Type != "PRIVATE KEY" {
		panic("block is not a private key or is nil")
	}

	pk, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	iat := time.Now()
	exp := iat.Add(3600 * time.Second)

	claims := jwt.StandardClaims{
		Audience:  "https://firestore.googleapis.com/",
		IssuedAt:  iat.Unix(),
		ExpiresAt: exp.Unix(),
		Issuer:    sv.ClientEmail,
		Subject:   sv.ClientEmail,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = sv.PrivateKeyId
	ss, err := token.SignedString(pk)
	if err != nil {
		panic(err)
	}
	fmt.Printf("------------------------------------\n")
	fmt.Printf("%+v\n", ss)
	fmt.Printf("------------------------------------\n")

	TryToGetAccessToken(ss)
}

type AccessTokenRequestBody struct {
	GrantType          string `json:"grantType"`
	Audience           string `json:"audience"`
	Scope              string `json:"scope"`
	RequestedTokenType string `json:"requestedTokenType"`
	SubjectToken       string `json:"subjectToken"`
	SubjectTokenType   string `json:"subjectTokenType"`
	Options            string `json:"options"`
}

func TryToGetAccessToken(token string) {
	client := http.DefaultClient

	const host = "https://sts.googleapis.com"
	const endpoint = host + "/v1/token"
	reqBody := AccessTokenRequestBody{
		GrantType:          "urn:ietf:params:oauth:grant-type:token-exchange",
		Audience:           host,
		Scope:              gmail.GmailReadonlyScope,
		RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		SubjectToken:       token,
		SubjectTokenType:   "urn:ietf:params:oauth:token-type:jwt",
		Options:            "",
	}

	jsonStr, err := json.Marshal(reqBody)
	if err != nil {
		panic(err)
	}

	resp, err := client.Post(endpoint, "application/json", strings.NewReader(string(jsonStr)))
	if err != nil {
		panic(err)
	}

	fmt.Printf(">>>>>>>>>>>>>>> Response: %+v\n\n\n\n", *resp)
}

func (gs *gmailService) AuthenticateService() {
	defer GetCredentialsFromServiceAccount()

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

	gs.srv = srv
}

func processAndGatherMessages(srv *gmail.Service, in <-chan string, out chan<- *gmail.Message) {
	const concurrentJobs = 10
	const user = "me"
	const msgFormat = "full"

	var wg sync.WaitGroup
	wg.Add(concurrentJobs)
	for i := 0; i < concurrentJobs; i++ {
		go func() {
			for msgId := range in {
				msg, _ := srv.Users.Messages.Get(user, msgId).Format(msgFormat).Do()
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

func concatFilters(filters []types.Filter) string {
	res := make([]string, len(filters))
	for i, filter := range filters {
		res[i] = fmt.Sprintf("%s:%s", filter.Type, filter.Value)
	}
	return strings.Join(res, " ")
}

func getMessageList(srv *gmail.Service, msgIdCh chan<- string, filters []types.Filter) {
	const user = "me"
	concatedFilters := concatFilters(filters)

	finished := false
	var msgs *gmail.ListMessagesResponse
	var err error
	var nextPageToken string
	for !finished {
		msgs, err = srv.Users.Messages.List(user).Q(concatedFilters).PageToken(nextPageToken).Do()
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

func convertHeadersToMap(headers []*gmail.MessagePartHeader) map[string]string {
	dict := map[string]string{}

	for _, header := range headers {
		key := header.Name
		value := header.Value
		dict[key] = value
	}

	return dict
}

// func convertBase64ToText(encoded string) (string, error) {
// 	decoded, err := base64.StdEncoding.DecodeString(encoded)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	decodedString := string(decoded)
// 	return decodedString, nil
// }
//
// func getTextPartsFromBody(parts []*gmail.MessagePart) []string {
// 	const mimeType = "text/html" // TODO change to various mimeTypes with a Set
// 	var res []string
//
// 	for _, part := range parts {
// 		if part.MimeType == mimeType {
// 			text, err := convertBase64ToText(part.Body.Data)
// 			if err != nil {
// 				log.Printf("Error decoding content: %s", err)
// 				continue
// 			}
//
// 			res = append(res, text)
// 		}
// 	}
//
// 	return res
// }

func extractMailMessageFromGmailMessage(msg *gmail.Message) types.Message {
	headers := convertHeadersToMap(msg.Payload.Headers)

	mailMsg := types.Message{
		Id:      msg.Id,
		Date:    headers["Date"],
		From:    headers["From"],
		To:      headers["To"],
		Subject: headers["Subject"],
		// Body: getTextPartsFromBody(msg.Payload.Parts),
		Body: []string{msg.Snippet}, // FIXME in the meantime with the snippet
	}

	return mailMsg
}

func (gs *gmailService) GetMessages(filters []types.Filter) []types.Message {
	in := make(chan string)
	out := make(chan *gmail.Message)

	go getMessageList(gs.srv, in, filters)
	processAndGatherMessages(gs.srv, in, out)

	var messages []types.Message
	for msg := range out {
		// fmt.Println(msg)
		if msg != nil {
			mailMsg := extractMailMessageFromGmailMessage(msg)
			messages = append(messages, mailMsg)
		}
	}

	return messages
}
