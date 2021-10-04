require (
	cloud.google.com/go v0.84.0 // indirect
	github.com/Philanthropists/toshl-email-autosync/mail v0.0.0-20210610050019-2b27ba4dc3ab
	github.com/Philanthropists/toshl-email-autosync/mail/gmail v0.0.0-20210610050019-2b27ba4dc3ab
	github.com/Philanthropists/toshl-email-autosync/market/investment_fund/bancolombia v0.0.0-20210610050019-2b27ba4dc3ab
	github.com/Philanthropists/toshl-email-autosync/market/rapidapi v0.0.0-20210610050019-2b27ba4dc3ab
	github.com/Philanthropists/toshl-go v0.1.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/oauth2 v0.0.0-20210615190721-d04028783cf1 // indirect
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22 // indirect
	google.golang.org/genproto v0.0.0-20210617175327-b9e0b3197ced // indirect
)

module github.com/Philanthropists/toshl-email-autosync

go 1.16

replace github.com/Philanthropists/toshl-email-autosync/mail => ./mail

replace github.com/Philanthropists/toshl-email-autosync/mail/gmail => ./mail/gmail

replace github.com/Philanthropists/toshl-email-autosync/market => ./market

replace github.com/Philanthropists/toshl-email-autosync/market/rapidapi => ./market/rapidapi

replace github.com/Philanthropists/toshl-email-autosync/market/investment_fund/bancolombia => ./market/investment-fund/bancolombia
