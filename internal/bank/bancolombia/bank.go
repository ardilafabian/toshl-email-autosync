package bancolombia

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	imaptypes "github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/common"
	synctypes "github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
)

type Bancolombia struct {
}

func (b Bancolombia) FilterMessage(msg imaptypes.Message) bool {
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

var regexpMap = map[string]*regexp.Regexp{
	"pago":          regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) a (?P<place>.+) desde (?:cta|T\.CRED) \*(?P<account>\d{4})\.`),
	"compra":        regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) en (?P<place>.+)\..+T\.(?:Cred|Deb) \*(?P<account>\d{4})\.`),
	"transferencia": regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) desde cta \*(?P<account>\d{4}).+cta (?P<place>\d{11,16})\.`),
}

func (b Bancolombia) ExtractTransactionInfoFromMessage(msg imaptypes.Message) (*synctypes.TransactionInfo, error) {
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

	result := common.ExtractFieldsStringWithRegexp(text, selectedRegexp)

	if !common.ContainsAllRequiredFields(result) {
		return nil, fmt.Errorf("message does not contain all required fields - result [%+v]", result)
	}
	value, err := getValueFromText(result["value"])
	if err != nil {
		return nil, err
	}

	return &synctypes.TransactionInfo{
		MsgId:   msg.SeqNum,
		Type:    result["type"],
		Place:   result["place"],
		Value:   value,
		Account: result["account"],
		Date:    msg.Envelope.Date,
	}, nil
}

// This would be way easier if Bancolombia had a consistent use of commas and dots inside the currency
var currencyRegexp = regexp.MustCompile(`^(?P<integer>[0-9\.,]+)(?P<decimal>\d{2})$`)

func getValueFromText(s string) (synctypes.Currency, error) {
	if !currencyRegexp.MatchString(s) {
		return synctypes.Currency{}, fmt.Errorf("string [%s] does not match regex [%s]", s, currencyRegexp.String())
	}

	res := common.ExtractFieldsStringWithRegexp(s, currencyRegexp)
	integer, ok := res["integer"]
	if !ok {
		return synctypes.Currency{}, fmt.Errorf("string [%s] should have an integer part", s)
	}

	decimal, ok := res["decimal"]
	if !ok {
		return synctypes.Currency{}, fmt.Errorf("string [%s] should have a decimal part", s)
	}

	integer = strings.ReplaceAll(integer, ",", "")
	integer = strings.ReplaceAll(integer, ".", "")
	valueStr := integer + "." + decimal
	value, err := strconv.ParseFloat(valueStr, 64)

	var currency synctypes.Currency
	currency.Code = "COP"
	currency.Rate = &value

	return currency, err
}
