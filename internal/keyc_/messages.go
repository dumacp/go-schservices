package keyc

import "net/http"

type MsgTick struct{}
type MsgGetClient struct{}
type MsgClient struct {
	Client *http.Client
}
