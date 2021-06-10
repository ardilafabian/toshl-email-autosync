require (
	cloud.google.com/go v0.83.0 // indirect
	github.com/Philanthropists/toshl-email-autosync/mail v0.0.0-20210606051336-f77707c2b291
	github.com/Philanthropists/toshl-email-autosync/mail/gmail v0.0.0-20210606051336-f77707c2b291
	github.com/Philanthropists/toshl-email-autosync/market/investment_fund/bancolombia v0.0.0-00010101000000-000000000000
	github.com/Philanthropists/toshl-email-autosync/market/rapidapi v0.0.0-00010101000000-000000000000
	github.com/andreagrandi/toshl-go v0.0.0-20161227205856-112e163a45ca
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/sys v0.0.0-20210603125802-9665404d3644 // indirect
	google.golang.org/genproto v0.0.0-20210604141403-392c879c8b08 // indirect
)

module github.com/Philanthropists/toshl-email-autosync

go 1.16

replace github.com/Philanthropists/toshl-email-autosync/mail => ./mail

replace github.com/Philanthropists/toshl-email-autosync/mail/gmail => ./mail/gmail

replace github.com/Philanthropists/toshl-email-autosync/market => ./market

replace github.com/Philanthropists/toshl-email-autosync/market/rapidapi => ./market/rapidapi

replace github.com/Philanthropists/toshl-email-autosync/market/investment_fund/bancolombia => ./market/investment_fund/bancolombia
