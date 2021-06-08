package mail

const (
	FromFilter = "from"
	AfterFilter = "after"
)

type FilterType string

type Filter struct {
	Type FilterType
	Value string
}

type Message struct {
	Id string
	Date string
	From string
	To string
	Subject string
	Body []string
}

type Service interface {
	AuthenticateService()
	GetMessages(filters []Filter) []Message
}