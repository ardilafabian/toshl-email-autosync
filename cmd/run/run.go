package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Philanthropists/toshl-email-autosync/internal/mail/imap"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/market/investment-fund/bancolombia"
	"github.com/Philanthropists/toshl-email-autosync/internal/market/rapidapi"
	"github.com/Philanthropists/toshl-email-autosync/internal/toshl_helper"

	"github.com/Philanthropists/toshl-go"
)

// func getMail() {
// 	service := gmail.GetGmailService()
// 	service.AuthenticateService()
//
// 	filters := []types.Filter{
// 		{
// 			Type:  types.FromFilter,
// 			Value: "alertasynotificaciones@bancolombia.com.co",
// 		},
// 		{
// 			Type:  types.AfterFilter,
// 			Value: "2020/01/05",
// 		},
// 	}
//
// 	for _, msg := range service.GetMessages(filters) {
// 		fmt.Println(msg)
// 	}
// }

//go:embed .auth.json
var rawAuth []byte

type Auth struct {
	Addr     string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var auth Auth

func GetStock() {
	api := rapidapi.RapidAPI{}
	err := api.GetCredentialsFromFile("rapidapi-keys.json")
	if err != nil {
		log.Fatal(err)
	}

	value, err := api.GetCurrentValue(rapidapi.USDCOP)
	if err != nil {
		log.Println("Error getting stock: ", err)
	}

	fmt.Printf("Current USD/COP value: %f\n", value)
}

func GetInvestmentFunds() {
	const fundName = "Renta Sostenible Global"
	list, err := bancolombia.GetAvailableInvestmentFundsBasicInfo()
	if err != nil {
		log.Println(err)
	}

	var fundId bancolombia.InvestmentFundId
	found := false
	for _, fundInfo := range list {
		if fundInfo.Name == fundName {
			fundId = fundInfo.Nit
			found = true
			break
		}
	}

	if !found {
		log.Printf("fund name [%s] not found in list", fundName)
		return
	}

	fund, err := bancolombia.GetInvestmentFundById(fundId)
	if err != nil {
		log.Println(err)
	}

	log.Printf("%+v", fund)
}

func GetToshlInfo() {
	token := toshl_helper.GetDefaultToshlToken()
	client := toshl.NewClient(token, nil)

	accounts, err := client.Accounts(nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%+v\n", accounts)
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

		shouldProcess := strings.Contains(text, "Pago")
		shouldProcess = shouldProcess || strings.Contains(text, "Compra")
		shouldProcess = shouldProcess || strings.Contains(text, "Transferencia")

		keep = shouldProcess
	}

	return keep
}

func GetLastProcessedDate() time.Time {
	// TODO get date from DynamoDB
	return time.Date(2021, time.September, 1, 0, 0, 0, 0, time.UTC)
}

func GetEmailFromBancolombia(mailClient imap.MailClient) ([]imap.Message, error) {
	const inboxMailbox = "INBOX"

	// mailboxes, err := mailClient.GetMailBoxes()
	// if err != nil {
	// 	return nil, err
	// }

	since := GetLastProcessedDate()
	messages, err := mailClient.GetMessages(inboxMailbox, since, filterFnc)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

type Currency struct {
	toshl.Currency
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
	"Pago":          regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) a (?P<place>.+) desde cta \*(?P<account>\d{4})\.`),
	"Compra":        regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) en (?P<place>.+)\..+T\.Cred \*(?P<account>\d{4})\.`),
	"Transferencia": regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) desde cta \*(?P<account>\d{4}).+(?P<place>\d{16})\.`),
}

func ExtractTransactionInfoFromMessages(msgs []imap.Message) ([]*TransactionInfo, error) {
	var transactions []*TransactionInfo
	for _, msg := range msgs {
		t, err := extractTransactionInfoFromMessage(msg)
		if err == nil {
			transactions = append(transactions, t)
		}
	}

	return transactions, nil
}

func extractTransactionInfoFromMessage(msg imap.Message) (*TransactionInfo, error) {
	text := string(msg.RawBody)

	var selected string
	for key := range regexpMap {
		if strings.Contains(text, key) {
			selected = key
			break
		}
	}

	if selected == "" {
		return nil, errors.New("message does not match any transaction type case")
	}

	selectedRegexp := regexpMap[selected]

	result := extractFieldsStringWithRegexp(text, selectedRegexp)

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
		if i != 0 && name != "" {
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
	currency.Rate = value

	return currency, err
}

func getAuth() {
	err := json.Unmarshal(rawAuth, &auth)
	if err != nil {
		panic(err)
	}
}

type Options struct {
	DryRun bool
	Debug  bool
}

func getOptions() Options {
	defer flag.Parse()

	var options Options

	flag.BoolVar(&options.DryRun, "dryRun", false, "Tell what will happen but not execute")
	flag.BoolVar(&options.Debug, "debug", false, "Output debug output")

	return options
}

func main() {
	_ = getOptions()

	getAuth()
	mailClient, err := imap.GetMailClient(auth.Addr, auth.Username, auth.Password)
	if err != nil {
		panic(err)
	}
	defer mailClient.Logout()

	msgs, err := GetEmailFromBancolombia(mailClient)
	if err != nil {
		panic(err)
	}

	transactions, _ := ExtractTransactionInfoFromMessages(msgs)

	for _, t := range transactions {
		log.Printf("%+v", t)
	}
}
