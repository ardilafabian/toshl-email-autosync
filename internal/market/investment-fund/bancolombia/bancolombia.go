package bancolombia

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/currency"
)

const (
	host          = "https://valores.grupobancolombia.com"
	fundsListPath = "consultarFondosInversion/rest/servicio/consultarListaFondos"
	fundById      = "consultarFondosInversion/rest/servicio/buscarInformacionFondo"
	copUnitCode   = "COP"
)

var copUnit currency.Unit

func init() {
	var err error
	copUnit, err = currency.ParseISO(copUnitCode)
	if err != nil {
		panic(err)
	}
}

type money currency.Amount

func (v *money) UnmarshalJSON(b []byte) error {
	cleaned := strings.Trim(string(b), "\"$")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return err
	}

	*v = money(copUnit.Amount(value))
	return nil
}

func (v money) String() string {
	return fmt.Sprintf("%s", currency.Amount(v))
}

type percentage float64

func (v *percentage) UnmarshalJSON(b []byte) error {
	cleaned := strings.Trim(string(b), "\"%")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return err
	}

	*v = percentage(value)
	return nil
}

type date time.Time

func (v *date) UnmarshalJSON(b []byte) error {
	const dateFormat = "20060102"

	cleaned := strings.Trim(string(b), "\"")
	timeDate, err := time.Parse(dateFormat, cleaned)
	if err != nil {
		return err
	}
	*v = date(timeDate)
	return nil
}

func (v date) String() string {
	return time.Time(v).String()
}

type InvestmentFundId string

type InvestmentFundBasicInfo struct {
	Nit  InvestmentFundId `json:"nit"`
	Name string           `json:"nombre"`
}

type InvestmentFund struct {
	InvestmentFundBasicInfo
	Score         string `json:"calificacion"`
	Term          string `json:"plazo"`
	UnitValue     money  `json:"valorDeUnidad"`
	CurrentValue  money  `json:"valorEnPesos"`
	Profitability struct {
		Days struct {
			WeeklyPercentage   percentage `json:"semanal"`
			MonthlyPercentage  percentage `json:"mensual"`
			SemesterPercentage percentage `json:"semestral"`
		} `json:"dias"`
		Years struct {
			Current        percentage `json:"anioCorrido"`
			LastYear       percentage `json:"ultimoAnio"`
			LastTwoYears   percentage `json:"ultimos2Anios"`
			LastThreeYears percentage `json:"ultimos3Anios"`
		} `json:"anios"`
	} `json:"rentabilidad"`
	ClosingDate   date   `json:"fechaCierre"`
	Administrator string `json:"sociedadAdministradora"`
}

func getFormedURIWithPath(path string) string {
	return fmt.Sprintf("%s/%s", host, path)
}

func doGetRequest(url string) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func GetAvailableInvestmentFundsBasicInfo() ([]InvestmentFundBasicInfo, error) {
	url := getFormedURIWithPath(fundsListPath)
	body, err := doGetRequest(url)
	if err != nil {
		return nil, err
	}

	var funds []InvestmentFundBasicInfo
	err = json.Unmarshal(body, &funds)
	if err != nil {
		return nil, err
	}

	return funds, nil
}

func GetInvestmentFundById(fundId InvestmentFundId) (InvestmentFund, error) {
	url := getFormedURIWithPath(fundById) + "/" + string(fundId)
	body, err := doGetRequest(url)
	if err != nil {
		return InvestmentFund{}, err
	}

	var fund InvestmentFund
	err = json.Unmarshal(body, &fund)
	if err != nil {
		return InvestmentFund{}, err
	}

	return fund, nil
}
