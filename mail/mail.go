package mail

type Filter string

type Message struct {
	Id string
	From string
	To string
	Subject string
	Body []string
}

type Service interface {
	AuthenticateService()
	GetMessages([]Filter) []Message
}