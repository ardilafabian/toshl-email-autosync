package toshl

import (
	"github.com/Philanthropists/toshl-go"
)

type Account struct {
	toshl.Account
}

type Entry struct {
	toshl.Entry
}

type Category struct {
	toshl.Category
}

type ApiClient interface {
	GetAccounts() ([]Account, error)
	CreateEntry(entry *Entry) error
	GetCategories() ([]Category, error)
	CreateCategory(category *Category) error
}

func NewApiClient(token string) ApiClient {
	return &clientImpl{client: toshl.NewClient(token, nil)}
}

type clientImpl struct {
	client *toshl.Client
}

func (c clientImpl) GetCategories() ([]Category, error) {
	categories, err := c.client.Categories(nil)
	if err != nil {
		return nil, err
	}

	var nCategories []Category
	for _, category := range categories {
		cat := Category{category}
		nCategories = append(nCategories, cat)
	}

	return nCategories, nil
}

func (c clientImpl) CreateCategory(category *Category) error {
	if err := c.client.CreateCategory(&category.Category); err != nil {
		return err
	}
	return nil
}

func (c clientImpl) CreateEntry(entry *Entry) error {
	if err := c.client.CreateEntry(&entry.Entry); err != nil {
		return err
	}
	return nil
}

func (c clientImpl) GetAccounts() ([]Account, error) {
	accounts, err := c.client.Accounts(nil)
	if err != nil {
		return nil, err
	}

	var nAccounts []Account
	for _, account := range accounts {
		nAccount := Account{account}
		nAccounts = append(nAccounts, nAccount)
	}

	return nAccounts, nil
}
