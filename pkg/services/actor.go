package params

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-params/internal/app"
)

func Actor(id string, discovery actor.Actor) actor.Actor {
	return app.NewActor(id, discovery)
}
