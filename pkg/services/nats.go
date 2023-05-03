package params

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-params/internal/nats"
)

func NatsActor(id string, discovery actor.Actor) actor.Actor {
	return nats.NewActor(id, discovery)
}
