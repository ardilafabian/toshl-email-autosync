module github.com/Philanthropists/toshl-email-autosync/mail/gmail

go 1.16

require (
	github.com/Philanthropists/toshl-email-autosync/mail v0.0.0-20210606051336-f77707c2b291
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	google.golang.org/api v0.47.0
)

replace github.com/Philanthropists/toshl-email-autosync/mail => ../
