package nats

import (
	"encoding/json"
	"fmt"

	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
)

func Parse(m []byte) interface{} {
	ev := &gwiotmsg.DiscoveryResponse{}

	if err := json.Unmarshal(m, ev); err != nil {
		return fmt.Errorf("error: %s", err)
	}
	return ev
}
