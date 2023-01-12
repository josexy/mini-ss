package auth

import (
	"net/url"
)

type Auth struct {
	info *url.Userinfo
}

func NewAuth(username, password string) *Auth {
	if username == "" {
		return nil
	}
	auth := new(Auth)
	if password != "" {
		auth.info = url.UserPassword(username, password)
	} else {
		auth.info = url.User(username)
	}
	return auth
}

func (a *Auth) UserInfo() *url.Userinfo {
	if a == nil {
		return nil
	}
	return a.info
}

func (a *Auth) Username() string {
	return a.info.Username()
}

func (a *Auth) Password() string {
	p, _ := a.info.Password()
	return p
}

func (a *Auth) String() string {
	return a.info.String()
}

func (a *Auth) Validate(username, password string) bool {
	p, has := a.info.Password()
	if has {
		return username == a.info.Username() && password == p
	}
	return username == a.info.Username()
}
