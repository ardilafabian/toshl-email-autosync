package market

type Service interface {
	GetCurrentValue(symbol string) (float64, error)
}
