package app

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/dumacp/go-actors/database"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/api/services"
	"github.com/dumacp/go-schservices/internal/nats"
	"github.com/dumacp/go-schservices/internal/utils"
)

var TIMEOUT = 3 * time.Minute

const (
	DATABASE_PATH = "/SD/bboltdb/servicesdb"
)

type Actor struct {
	id             string
	evs            *eventstream.EventStream
	contxt         context.Context
	currentService *services.ScheduleService
	dataActor      actor.Actor
	db             *actor.PID
	cancel         func()
}

func NewActor(id string, data actor.Actor) actor.Actor {
	if len(id) == 0 {
		id = utils.Hostname()
	}
	a := &Actor{id: id}
	a.dataActor = data
	a.evs = &eventstream.EventStream{}
	return a
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
	fmt.Printf("message: %q --> %q, %T (%s)\n", func() string {
		if ctx.Sender() == nil {
			return ""
		} else {
			return ctx.Sender().GetId()
		}
	}(), ctx.Self().GetId(), ctx.Message(), ctx.Message())
	switch msg := ctx.Message().(type) {
	case *actor.Started:

		db, err := database.Open(ctx, DATABASE_PATH)
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panicf("open database (%s) error: %s", DATABASE_PATH, err)
		}

		a.db = db.PID()
		if a.dataActor != nil {
			if _, err := ctx.SpawnNamed(actor.PropsFromFunc(a.dataActor.Receive), "schsvc-data-actor"); err != nil {
				time.Sleep(3 * time.Second)
				logs.LogError.Panicf("open nats-actor error: %s", err)
			}
		} else {
			logs.LogWarn.Printf("without props for data")
		}

		conx, cancel := context.WithCancel(context.TODO())
		a.cancel = cancel
		a.contxt = conx
		go tick(conx, ctx, TIMEOUT)
		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
	// case *gwiotmsg.Disconnected:
	// 	logs.LogWarn.Printf("error disconnect: %s", msg.Error)
	// 	a.evs.Publish(&services.StatusSch{
	// 		State: 0,
	// 	})
	// 	if ctx.Parent() != nil {
	// 		ctx.Request(ctx.Parent(), &services.StatusSch{
	// 			State: 0,
	// 		})
	// 	}
	case *services.RequestStatusSch:

	case *nats.MsgStatus:
		logs.LogInfo.Printf("data connected: %v", msg.State)
		state := func() int32 {
			if msg.State {
				return 1
			}
			return 0
		}()
		a.evs.Publish(&services.StatusSch{
			State: state,
		})
		if ctx.Parent() != nil {
			ctx.Request(ctx.Parent(), &services.StatusSch{
				State: state,
			})
		}
	case *actor.Stopping:
		if a.cancel != nil {
			a.cancel()
		}
	case *MsgTick:
	case *services.Mods:

		ss := msg.GetUpdates()
		// sort.SliceStable(ss, func(i, j int) bool {
		// 	return ss[i].GetScheduleDateTime() < ss[j].GetScheduleDateTime()
		// })
		for _, update := range ss {

			fmt.Printf("//////////////**************** update: %v\n", update)
			fmt.Printf("//////////////**************** state: %v - %s\n", update.GetState(), update.GetState())
			fmt.Printf("//////////////**************** timingState: %v - %s\n",
				update.GetCheckpointTimingState().GetState(), update.GetCheckpointTimingState().GetState())
			switch update.State {
			default:
				a.evs.Publish(&services.UpdateServiceMsg{
					Update: update,
				})
				if ctx.Parent() != nil {
					ctx.Request(ctx.Parent(), &services.UpdateServiceMsg{
						Update: update,
					})
				}
				// default:
				// 	a.evs.Publish(&services.ServiceMsg{
				// 		Update: update,
				// 	})
				// 	if ctx.Parent() != nil {
				// 		ctx.Request(ctx.Parent(), &services.ServiceMsg{
				// 			Update: update,
				// 		})
				// 	}
				// case services.State_STARTED.String(),
				// 	services.State_READY_TO_START.String(),
				// 	services.State_WAITING_TO_ARRIVE_TO_STARTING_POINT.String():
				// 	serviceTime := time.UnixMilli(update.GetScheduleDateTime())
				// 	if time.Until(serviceTime) > 0 ||
				// 		time.Since(serviceTime) < 20*time.Minute ||
				// 		time.Since(serviceTime) < 60*time.Minute && update.State == services.State_STARTED.String() {
				// 		a.evs.Publish(update)
				// 		if ctx.Parent() != nil {
				// 			ctx.Request(ctx.Parent(), update)
				// 		}
				// 	}
				// case services.State_SCHEDULED.String():
				// 	serviceTime := time.UnixMilli(update.GetScheduleDateTime())
				// 	if time.Until(serviceTime) <= 0 {
				// 		break
				// 	}
				// 	a.evs.Publish(update)
				// 	if ctx.Parent() != nil {
				// 		ctx.Request(ctx.Parent(), update)
				// 	}
				// case services.State_CANCELLED.String(),
				// 	services.State_ABORTED.String(), services.State_ENDED.String():
				// 	a.evs.Publish(update)
				// 	if ctx.Parent() != nil {
				// 		ctx.Request(ctx.Parent(), update)
				// 	}
			}
		}

		sa := msg.GetAdditions()
		// sort.SliceStable(ss, func(i, j int) bool {
		// 	return ss[i].GetScheduleDateTime() < ss[j].GetScheduleDateTime()
		// })
		for _, update := range sa {
			fmt.Printf("//////////////**************** addition: %v\n", update)
			a.evs.Publish(&services.UpdateServiceMsg{
				Update: update,
			})
			if ctx.Parent() != nil {
				ctx.Request(ctx.Parent(), &services.UpdateServiceMsg{
					Update: update,
				})
			}
		}

		sr := msg.GetRemovals()
		// sort.SliceStable(ss, func(i, j int) bool {
		// 	return ss[i].GetScheduleDateTime() < ss[j].GetScheduleDateTime()
		// })
		if len(sr) > 0 {
			go func() {
				// delay remove services
				time.Sleep(3 * time.Second)
				for _, update := range sr {

					fmt.Printf("//////////////**************** remove: %v\n", update)
					a.evs.Publish(&services.RemoveServiceMsg{
						Update: update,
					})
					if ctx.Parent() != nil {
						ctx.Request(ctx.Parent(), &services.RemoveServiceMsg{
							Update: update,
						})
					}
				}
			}()
		}

	case *services.Snapshot:
		ss := msg.GetScheduledServices()
		fmt.Printf("////// services len: %d\n", len(ss))
		// sort.SliceStable(ss, func(i, j int) bool {
		// 	return ss[i].GetScheduleDateTime() < ss[j].GetScheduleDateTime()
		// })
		// for _, update := range ss {
		// 	fmt.Printf("//////////////**************** state: %v - %s\n", update.GetState(), update.GetState())
		// 	a.evs.Publish(update)
		// 	if ctx.Parent() != nil {
		// 		ctx.Request(ctx.Parent(), &services.RemoveServiceMsg{
		// 			Update: update,
		// 		})
		// 	}
		// }
		a.evs.Publish(&services.ServiceAllMsg{
			Updates: ss,
		})
		if ctx.Parent() != nil {
			ctx.Request(ctx.Parent(), &services.ServiceAllMsg{
				Updates: ss,
			})
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
