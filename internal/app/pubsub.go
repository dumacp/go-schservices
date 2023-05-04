package app

import (
	"encoding/json"

	"github.com/dumacp/go-schservices/api/services"
)

const (
	SUBS_TOPIC = "SCHSERVICES/subscribe"
)

type ExternalSubscribe struct {
	ID     string
	Addres string
}

func Discover(msg []byte) interface{} {

	subs := new(services.DiscoverSch)

	if err := json.Unmarshal(msg, subs); err != nil {
		return err
	}

	return subs
}
