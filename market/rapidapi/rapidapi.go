package rapidapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const StockSummaryURI = "https://apidojo-yahoo-finance-v1.p.rapidapi.com/stock/v2/get-summary?region=US&symbol=%s"

type RapidAPI struct {
	Uri    string
	Header struct {
		Key  string `json:"key"`
		Host string `json:"host"`
	}
}

const (
	USDCOP = "COP%3DX"
)

type stockValue struct {
	Price struct {
		RegularMarketOpen struct {
			Raw float64 `json:"raw"`
		} `json:"regularMarketOpen"`
	} `json:"price"`
}

func (api *RapidAPI) GetCredentialsFromFile(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to read API keys from file: %v", err)
		return errors.New(errMsg)
	}

	err = json.Unmarshal(b, &api.Header)
	if err != nil {
		return err
	}

	return nil
}

func (api *RapidAPI) GetCurrentValue(symbol string) (float64, error) {
	const HeaderKey = "x-rapidapi-key"
	const HeaderHost = "x-rapidapi-host"

	if api.Uri == "" {
		api.Uri = StockSummaryURI
	}

	url := fmt.Sprintf(api.Uri, symbol)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Add(HeaderKey, api.Header.Key)
	req.Header.Add(HeaderHost, api.Header.Host)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	var value stockValue
	err = json.NewDecoder(res.Body).Decode(&value)
	if err != nil {
		return 0, err
	}

	return value.Price.RegularMarketOpen.Raw, nil
}
