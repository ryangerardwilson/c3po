package gmail

import (
	"github.com/ryangerardwilson/c3po/gmail/internal/api"
	"github.com/ryangerardwilson/c3po/gmail/internal/config"
	"github.com/ryangerardwilson/c3po/gmail/internal/oauth"
)

func (e Env) service(acct config.Account) (Service, error) {
	if e.Box != nil {
		return e.Box(acct)
	}
	opt := oauth.Options{
		HTTP: e.HTTP, Paths: e.paths(), Now: e.Now,
		OpenBrowser: e.OpenBrowser, Out: e.Stdout,
	}
	return &api.Client{HTTP: e.HTTP, Access: func() (string, error) {
		return opt.AccessToken(acct.Email, acct.ClientSecretFile)
	}}, nil
}
