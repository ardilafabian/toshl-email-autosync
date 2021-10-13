package types

const (
	FromFilter FilterType = iota
	AfterFilter
)

type FilterType int64

func (f FilterType) String() string {
	switch f {
	case FromFilter:
		return "from"
	case AfterFilter:
		return "after"
	}

	return "unknown"
}

type Filter struct {
	Type  FilterType
	Value string
}

type Message struct {
	Id      string
	Date    string
	From    string
	To      string
	Subject string
	Body    []string
}

type Service interface {
	AuthenticateService()
	GetMessages(filters []Filter) []Message
}
