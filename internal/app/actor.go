package app

import (
	"context"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/api/pb/serviceschmsg"
	"github.com/dumacp/go-schservices/internal/utils"
)

var TIMEOUT = 3 * time.Minute

type Actor struct {
	id     string
	evs    *eventstream.EventStream
	contxt context.Context
	cancel func()
}

func NewActor(id string) actor.Actor {
	if len(id) == 0 {
		id = utils.Hostname()
	}
	return &Actor{id: id}
}

func subscribe(ctx actor.Context, evs *eventstream.EventStream) *eventstream.Subscription {
	rootctx := ctx.ActorSystem().Root
	pid := ctx.Sender()
	self := ctx.Self()

	fn := func(evt interface{}) {
		rootctx.RequestWithCustomSender(pid, evt, self)
	}
	sub := evs.Subscribe(fn)
	return sub
}

func (a *Actor) Receive(ctx actor.Context) {
	logs.LogBuild.Printf("Message arrived in %s: %s, %T, %s",
		ctx.Self().GetId(), ctx.Message(), ctx.Message(), ctx.Sender())
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		conx, cancel := context.WithCancel(context.TODO())
		a.cancel = cancel
		a.contxt = conx
		go tick(conx, ctx, TIMEOUT)
		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
	case *actor.Stopping:
		if a.cancel != nil {
			a.cancel()
		}
	case *MsgTick:
	case *serviceschmsg.ScheduleServices:
		var prevScheduleService *serviceschmsg.ScheduleService
		for _, scheduleService := range msg.ScheduleServices {
			scheduleUnixTime := time.Unix(scheduleService.ScheduleDateTime, 0)
			delay := time.Until(scheduleUnixTime)
			if delay > 0 {
				go func(scheduleService *serviceschmsg.ScheduleService) {
					select {
					case <-time.After(delay):
						// Enviar el mensaje ScheduleService a los actores deseados
						// Aquí necesitas reemplazar "targetActorPID" por el PID del actor al que deseas enviar el mensaje
						a.evs.Publish(scheduleService)
					case <-a.contxt.Done():
					}
				}(scheduleService)
			} else {
				// Si el tiempo de programación ya pasó, enviar el mensaje inmediatamente
				prevScheduleService = scheduleService
			}
		}
		if prevScheduleService != nil {
			a.evs.Publish(prevScheduleService)
		}
	}
}

func tick(ctx context.Context, ctxactor actor.Context, timeout time.Duration) {
	rootctx := ctxactor.ActorSystem().Root
	self := ctxactor.Self()
	t0_1 := time.NewTimer(3000 * time.Millisecond)
	defer t0_1.Stop()
	t2 := time.NewTicker(30 * time.Second)
	defer t2.Stop()

	for {
		select {
		case <-t0_1.C:
			rootctx.Send(self, &MsgTriggerServices{})
		case <-t2.C:
			rootctx.Send(self, &MsgTriggerServices{})
		case <-ctx.Done():
			return
		}
	}
}
