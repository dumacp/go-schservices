package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/pkg/messages"
)

var TIMEOUT = 3 * time.Minute

type Actor struct {
	liveService       *messages.ScheduleService
	scheduledServices []*messages.ScheduleService
	evs               *eventstream.EventStream
	pidGetData        *actor.PID
	propsGetData      *actor.Props
	cancel            func()
	id                string
}

func NewActor(id string, props *actor.Props) (actor.Actor, error) {
	if props == nil {
		return nil, fmt.Errorf("props is nil")
	}
	return &Actor{id: id, propsGetData: props}, nil
}

func subscribeServices(ctx actor.Context, evs *eventstream.EventStream) *eventstream.Subscription {
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
		pid, err := ctx.SpawnNamed(a.propsGetData, a.id)
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panic(err)
		}
		a.pidGetData = pid
		conx, cancel := context.WithCancel(context.TODO())
		a.cancel = cancel
		go tick(conx, ctx, TIMEOUT)
		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
	case *actor.Stopping:
		a.cancel()
	case *MsgTick:
	case *messages.DiscoverSch:
		if len(msg.GetAddr()) <= 0 || len(msg.GetId()) <= 0 {
			break
		}
		pid := actor.NewPID(msg.GetAddr(), msg.GetId())
		ctx.Request(pid, &messages.DiscoverResponseSch{
			Id:   ctx.Self().GetId(),
			Addr: ctx.Self().GetAddress(),
		})

	case *MsgTriggerServices:

		func() {
			// fmt.Println("eval newest live")
			if newestService, ok := Next(a.scheduledServices, time.Now().Add(-1*time.Minute), time.Now().Add(1*time.Minute)); ok {
				if a.liveService.GetId() != newestService.GetId() {
					fmt.Printf("/////////////////////// id: %s !== %s\n", a.liveService.GetId(), newestService.GetId())
					ctx.Send(ctx.Self(), &MsgPublishServices{
						Data: newestService,
					})
					// fmt.Println("publish newest live")
				}
				return
			}
		}()
	case *MsgScheduleServiceData:
		if err := func() error {
			logs.LogBuild.Printf("Get response, GetScheduledServices: %v", msg.Data)

			if len(msg.Data) <= 0 {
				return errors.New("error empty ScheduledServices result")
			}
			sort.Slice(msg.Data, func(i, j int) bool {
				return msg.Data[j].GetScheduleDateTime() < msg.Data[i].GetScheduleDateTime()
			})

			a.scheduledServices = msg.Data
			return nil
		}(); err != nil {
			logs.LogWarn.Println(err)
			fmt.Printf("GetScheduledServices err: %s\n", err)
			return
		}

		if len(a.scheduledServices) > 0 {
			if data, err := json.Marshal(a.scheduledServices); err == nil {
				logs.LogInfo.Printf("ScheduledServices: %s", data)
			}
		}
	case *messages.SubscribeSch:
		if ctx.Sender() == nil {
			break
		}
		ctx.RequestWithCustomSender(ctx.Self(), &MsgSubscribeServices{}, ctx.Sender())
	case *MsgSubscribeServices:
		if a.evs == nil {
			a.evs = eventstream.NewEventStream()
		}
		subscribeServices(ctx, a.evs)
		if Valid(a.liveService) && ctx.Sender() != nil {
			svc := a.liveService
			ctx.Send(ctx.Sender(), &svc)
		}
	case *MsgPublishServices:
		if a.evs != nil {
			// fmt.Printf("%s: publish service: %v\n", time.Now().Format("15:04:05.000"), msg.Data)
			// fmt.Printf("%s: before service: %v\n", time.Now().Format("15:04:05.000"), a.liveService)
			a.evs.Publish(a.liveService)
		}
		fmt.Printf("%s: publish service: %v\n", time.Now().Format("15:04:05.000"), msg.Data)
		a.liveService = msg.Data

	case *messages.RequestStatusSch:
		if ctx.Sender() == nil {
			break
		}
		if len(a.scheduledServices) > 0 {
			ctx.Respond(&messages.StatusSch{State: 1})
		} else {
			ctx.Respond(&messages.StatusSch{State: 0})
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
