package sse

import "net/http"

type UserAuthenticator func(r *http.Request) (userKey string, err error)

type Options struct {
	NoXAccelBuffering  bool
	SendChanBufferSize uint
	UserAuthenticator  UserAuthenticator
}

func NewDefaultOptions() Options {
	return Options{
		NoXAccelBuffering:  true,
		SendChanBufferSize: 100,
		UserAuthenticator:  nil,
	}
}
