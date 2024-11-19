package services

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-schservices/internal/app"
)

func Actor(id, url string, discovery actor.Actor) actor.Actor {
	return app.NewActor(id, url, discovery)
}
