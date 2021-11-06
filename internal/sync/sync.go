package sync

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/Philanthropists/toshl-email-autosync/internal/dynamodb"
	"github.com/Philanthropists/toshl-email-autosync/internal/mail/imap"
	"github.com/Philanthropists/toshl-email-autosync/internal/toshl"
	"github.com/Philanthropists/toshl-email-autosync/internal/twilio"

	toshlclient "github.com/Philanthropists/toshl-go"
)

var localLocation *time.Location

func init() {
	var err error
	localLocation, err = time.LoadLocation("America/Bogota")
	if err != nil {
		panic(err)
	}
}

type Auth struct {
	Addr             string `json:"mail-addr"`
	Username         string `json:"mail-username"`
	Password         string `json:"mail-password"`
	ToshlToken       string `json:"toshl-token"`
	TwilioAccountSid string `json:"twilio-account-sid"`
	TwilioAuthToken  string `json:"twilio-auth-token"`
	TwilioFromNumber string `json:"twilio-from-number"`
	TwilioToNumber   string `json:"twilio-to-number"`
}

var filterFnc imap.Filter = func(msg imap.Message) bool {
	keep := true
	keep = keep && msg.Message != nil
	keep = keep && msg.Message.Envelope != nil
	if keep {
		keep = false
		for _, address := range msg.Message.Envelope.From {
			from := address.Address()
			if from == "alertasynotificaciones@notificacionesbancolombia.com" {
				keep = true
				break
			}
		}
	}

	if keep {
		text := string(msg.RawBody)
		lowerCaseText := strings.ToLower(text)

		shouldProcess := strings.Contains(lowerCaseText, "pago")
		shouldProcess = shouldProcess || strings.Contains(lowerCaseText, "compra")
		shouldProcess = shouldProcess || strings.Contains(lowerCaseText, "transferencia")

		keep = shouldProcess
	}

	return keep
}

func GetLastProcessedDate() time.Time {
	const dateField = "LastProcessedDate"
	const tableName = "toshl-data"
	defaultDate := time.Now().Add(-365 * 24 * time.Hour) // from 1 year in the past by default

	var selectedDate time.Time

	client, err := dynamodb.NewClient("us-east-1")
	if err != nil {
		log.Fatalf("error creating dynamodb client: %s", err)
	}

	res, err := client.Scan(tableName)
	if err != nil {
		selectedDate = defaultDate
		log.Printf("connect to dynamodb unsuccessfull: %s\n", err)
	} else if len(res) == 1 {
		resValue := res[0]
		value, ok := resValue[dateField]
		if ok {
			switch j := value.(type) {
			case string:
				selectedDate, err = time.Parse(time.RFC822Z, j)
				if err != nil {
					selectedDate = defaultDate
				}
			}
		} else {
			selectedDate = defaultDate
			log.Printf("%s field is not defined in item", dateField)
		}
	} else {
		selectedDate = defaultDate
		log.Printf("something is wrong, the len was not 1: [%+v]", res)
	}

	log.Printf("selected date: %s", selectedDate.Format(time.RFC822Z))

	return selectedDate
}

func GetEmailFromBancolombia(mailClient imap.MailClient) ([]imap.Message, error) {
	const inboxMailbox = "INBOX"

	since := GetLastProcessedDate()
	messages, err := mailClient.GetMessages(inboxMailbox, since, filterFnc)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

type Currency struct {
	toshlclient.Currency
}

type TransactionInfo struct {
	MsgId   uint32
	Type    string
	Place   string
	Value   Currency
	Account string
	Date    time.Time
}

var regexpMap = map[string]*regexp.Regexp{
	"pago":          regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) a (?P<place>.+) desde cta \*(?P<account>\d{4})\.`),
	"compra":        regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) en (?P<place>.+)\..+T\.Cred \*(?P<account>\d{4})\.`),
	"transferencia": regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) desde cta \*(?P<account>\d{4}).+(?P<place>\d{16})\.`),
}

func ExtractTransactionInfoFromMessages(msgs []imap.Message) ([]*TransactionInfo, int64, error) {
	var failures int64

	var transactions []*TransactionInfo
	for _, msg := range msgs {
		t, err := extractTransactionInfoFromMessage(msg)
		if err == nil {
			transactions = append(transactions, t)
		} else {
			log.Printf("Error processing message: %s", err)
			failures++
		}
	}

	return transactions, failures, nil
}

func extractTransactionInfoFromMessage(msg imap.Message) (*TransactionInfo, error) {
	text := string(msg.RawBody)
	lowerCaseText := strings.ToLower(text)

	var selected string
	for key := range regexpMap {
		if strings.Contains(lowerCaseText, key) {
			selected = key
			break
		}
	}

	if selected == "" {
		return nil, errors.New("message does not match any transaction type case")
	}

	selectedRegexp := regexpMap[selected]

	result := extractFieldsStringWithRegexp(text, selectedRegexp)

	log.Printf("Values: %+v\n", result)

	value, err := getValueFromText(result["value"])
	if err != nil {
		return nil, err
	}

	return &TransactionInfo{
		MsgId:   msg.SeqNum,
		Type:    result["type"],
		Place:   result["place"],
		Value:   value,
		Account: result["account"],
		Date:    msg.Envelope.Date,
	}, nil
}

func extractFieldsStringWithRegexp(s string, r *regexp.Regexp) map[string]string {
	match := r.FindStringSubmatch(s)
	result := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 && name != "" && i < len(match) {
			result[name] = match[i]
		}
	}

	return result
}

// This would be way easier if Bancolombia had a consistent use of commas and dots inside the currency
var currencyRegexp = regexp.MustCompile(`^(?P<integer>[0-9\.,]+)(?P<decimal>\d{2})$`)

func getValueFromText(s string) (Currency, error) {
	if !currencyRegexp.MatchString(s) {
		return Currency{}, fmt.Errorf("string [%s] does not match regex [%s]", s, currencyRegexp.String())
	}

	res := extractFieldsStringWithRegexp(s, currencyRegexp)
	integer, ok := res["integer"]
	if !ok {
		return Currency{}, fmt.Errorf("string [%s] should have an integer part", s)
	}

	decimal, ok := res["decimal"]
	if !ok {
		return Currency{}, fmt.Errorf("string [%s] should have a decimal part", s)
	}

	integer = strings.ReplaceAll(integer, ",", "")
	integer = strings.ReplaceAll(integer, ".", "")
	valueStr := integer + "." + decimal
	value, err := strconv.ParseFloat(valueStr, 64)

	var currency Currency
	currency.Code = "COP"
	currency.Rate = &value

	return currency, err
}

func getMappableAccounts(accounts []*toshl.Account) map[string]*toshl.Account {
	var exp = regexp.MustCompile(`^(?P<account>\d+) `)

	var mapping = make(map[string]*toshl.Account)
	for _, account := range accounts {
		name := account.Name
		result := extractFieldsStringWithRegexp(name, exp)
		fmt.Printf("name: %s - result: %s\n", name, result)
		if num, ok := result["account"]; ok {
			mapping[num] = account
		}
	}

	return mapping
}

func createInternalCategoryIfAbsent(toshlClient toshl.ApiClient) string {
	const categoryName = "PENDING"

	categories, err := toshlClient.GetCategories()
	if err != nil {
		panic(err)
	}

	for _, c := range categories {
		if c.Name == categoryName {
			return c.ID
		}
	}

	var cat toshl.Category
	cat.Name = categoryName
	cat.Type = "expense"

	err = toshlClient.CreateCategory(&cat)
	if err != nil {
		panic(err)
	}

	return cat.ID
}

func UpdateLastProcessedDate(failedTxs []*TransactionInfo) error {
	newDate := getEarliestDateFromTxs(failedTxs)

	const idField = "Id"
	const dateField = "LastProcessedDate"
	const tableName = "toshl-data"

	client, err := dynamodb.NewClient("us-east-1")
	if err != nil {
		log.Fatalf("error creating dynamodb client: %s", err)
	}

	key := map[string]dynamodb.AttributeValue{
		idField: {
			AttributeValue: &types.AttributeValueMemberN{Value: "1"},
		},
	}

	expressionAttributeValues := map[string]dynamodb.AttributeValue{
		":r": {
			AttributeValue: &types.AttributeValueMemberS{Value: newDate.Format(time.RFC822Z)},
		},
	}

	updateExpression := fmt.Sprintf("set %s = :r", dateField)

	err = client.UpdateItem(tableName, key, expressionAttributeValues, updateExpression)
	return err
}

func getEarliestDateFromTxs(txs []*TransactionInfo) time.Time {
	earliestDate := time.Now().Add(-24 * time.Hour)
	for _, tx := range txs {
		date := tx.Date
		if date.Before(earliestDate) {
			earliestDate = date
		}
	}

	return earliestDate
}

func ArchiveEmailsOfSuccessfulTransactions(mailClient imap.MailClient, successfulTransactions []*TransactionInfo) {
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

func CreateEntries(toshlClient toshl.ApiClient, transactions []*TransactionInfo, mappableAccounts map[string]*toshl.Account, internalCategoryId string) ([]*TransactionInfo, []*TransactionInfo) {
	const DateFormat = "2006-01-02"

	var successfulTransactions []*TransactionInfo
	var failedTransactions []*TransactionInfo
	for _, t := range transactions {
		account, ok := mappableAccounts[t.Account]
		if !ok {
			continue
		}

		var newEntry toshl.Entry
		newEntry.Amount = -*t.Value.Rate // negative because it is an expense
		newEntry.Currency = toshlclient.Currency{
			Code: "COP",
		}
		newEntry.Date = t.Date.In(localLocation).Format(DateFormat)
		description := fmt.Sprintf("** %s de %s", t.Type, t.Place)
		newEntry.Description = &description
		newEntry.Account = account.ID
		newEntry.Category = internalCategoryId

		err := toshlClient.CreateEntry(&newEntry)
		if err != nil {
			log.Printf("Failed to create entry for transaction [%+v | %+v]: %s\n", newEntry, t, err)
			failedTransactions = append(failedTransactions, t)
		} else {
			log.Printf("Created entry %+v sucessfully", newEntry)
			successfulTransactions = append(successfulTransactions, t)
		}
	}

	return successfulTransactions, failedTransactions
}

func SendNotifications(auth Auth, msg string) {
	accountSid := auth.TwilioAccountSid
	authToken := auth.TwilioAuthToken
	fromNumber := auth.TwilioFromNumber
	toNumber := auth.TwilioToNumber

	client, err := twilio.NewClient(accountSid, authToken)
	if err != nil {
		log.Printf("error: could not instantiate twilio client: %s", err)
		return
	}

	_, err = client.SendSms(fromNumber, toNumber, msg)
	if err != nil {
		log.Printf("error: an error ocurred when sending notification sms")
		return
	}
}

func Run(ctx context.Context, auth Auth) error {
	mailClient, err := imap.GetMailClient(auth.Addr, auth.Username, auth.Password)
	if err != nil {
		return err
	}
	defer mailClient.Logout()

	msgs, err := GetEmailFromBancolombia(mailClient)
	if err != nil {
		return err
	}

	transactions, failures, _ := ExtractTransactionInfoFromMessages(msgs)

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
	internalCategoryId := createInternalCategoryIfAbsent(toshlClient)

	accounts, err := toshlClient.GetAccounts()
	if err != nil {
		return err
	}

	for _, a := range accounts {
		log.Printf("Accounts %+v", a)
	}

	mappableAccounts := getMappableAccounts(accounts)

	for name, account := range mappableAccounts {
		log.Printf("Mappable accounts: [%s] : %+v", name, account)
	}

	successfulTxs, failedTxs := CreateEntries(toshlClient, transactions, mappableAccounts, internalCategoryId)

	ArchiveEmailsOfSuccessfulTransactions(mailClient, successfulTxs)

	msg := fmt.Sprintf("Synced transactions: %d sucessful - %d failed", len(successfulTxs), len(failedTxs))
	log.Println(msg)
	if len(successfulTxs) > 0 {
		SendNotifications(auth, msg)
	}

	if err := UpdateLastProcessedDate(failedTxs); err != nil {
		return fmt.Errorf("failed to update last processed date: %s", err)
	}

	return nil
}
