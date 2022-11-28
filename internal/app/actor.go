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
	liveService                       Service
	scheduledServices                 []Service
	liveExecutedServices              []Service
	evs                               *eventstream.EventStream
	pidGetData                        *actor.PID
	propsGetData                      *actor.Props
	cancel                            func()
	id                                string
	lastTimestampScheduledServices    int64
	lastTimestampLiveExecutedServices int64
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
		a.liveService = Service{}
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
	case *MsgKeycloak:
		if ctx.Sender() != nil {
			a.pidGetData = ctx.Sender()
		}
		ctx.Send(ctx.Self(), &MsgGetServices{})
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
			if newestService, ok := Next(a.liveExecutedServices, time.Now().Add(-1*time.Minute), time.Now().Add(1*time.Minute)); ok {
				if !a.liveService.Same(newestService) {
					ctx.Send(ctx.Self(), &MsgPublishServices{
						Data: newestService,
					})
					// fmt.Println("publish newest live")
				}
				return
			}
			// fmt.Println("eval newest schedule")
			if newestService, ok := Next(a.scheduledServices, time.Now(), time.Now().Add(3*time.Minute)); ok {
				if !a.liveService.Same(newestService) {
					ctx.Send(ctx.Self(), &MsgGetLiveExecutedServices{})
					// fmt.Println("publish newest schedule")
				}
			}

			newestScheduleService, _ := Newest(a.scheduledServices)
			newestLiveService, _ := Newest(a.liveExecutedServices)

			// fmt.Println("eval newest all")
			if newestService, ok := Next([]Service{newestScheduleService, newestLiveService},
				time.Now().Add(-120*time.Minute), time.Now().Add(-2*time.Minute)); ok {
				if !a.liveService.Same(newestService) {
					ctx.Send(ctx.Self(), &MsgPublishServices{
						Data: newestService,
					})
					// fmt.Println("publish newest all")
				}
				return
			}
		}()
	case *MsgGetServices:
		if msg.VerifyNil {
			if len(a.liveExecutedServices) > 0 || len(a.scheduledServices) > 0 {
				break
			}
		}
		if a.pidGetData == nil {
			break
		}
		ctx.Request(a.pidGetData, &MsgGetServiceData{})
	case *MsgServiceData:
		if err := func() error {
			logs.LogBuild.Printf("Get response, GetServices: %s", msg.Data)
			result := new(Services)
			if err := json.Unmarshal(msg.Data, result); err != nil {
				return err
			}
			if result.LastTimestampLiveExecutedServices > a.lastTimestampLiveExecutedServices {
				ctx.Send(ctx.Self(), &MsgGetLiveExecutedServices{})
			}
			if result.LastTimestampScheduledServices > a.lastTimestampScheduledServices {
				ctx.Send(ctx.Self(), &MsgGetScheduledServices{})
			}
			return nil
		}(); err != nil {
			logs.LogError.Println(err)
			fmt.Printf("GetServices err: %s\n", err)
		}
	case *MsgGetScheduledServices:
		if msg.VerifyNil {
			if len(a.scheduledServices) > 0 {
				break
			}
		}
		if a.pidGetData == nil {
			break
		}
		ctx.Request(a.pidGetData, &MsgGetScheduleServiceData{})
	case *MsgScheduleServiceData:
		if err := func() error {
			logs.LogBuild.Printf("Get response, GetScheduledServices: %s", msg.Data)
			result := new(Services)
			if err := json.Unmarshal(msg.Data, result); err != nil {
				return err
			}
			a.lastTimestampScheduledServices = result.LastTimestampScheduledServices
			if len(result.ScheduledServices) <= 0 {
				return errors.New("error empty ScheduledServices result")
			}
			sort.Slice(result.ScheduledServices, func(i, j int) bool {
				return result.ScheduledServices[j].Timestamp < result.ScheduledServices[i].Timestamp
			})
			a.scheduledServices = make([]Service, 0)
			for _, v := range result.ScheduledServices {
				if (v == Service{}) {
					continue
				}
				if v.TimeService <= v.Timestamp {
					v.TimeService = v.Timestamp + 120*60*1*1000 // 120 minutos
				}
				a.scheduledServices = append(a.scheduledServices, v)
				if v.Timestamp < time.Now().Add(-24*time.Hour).UnixMilli() {
					break
				}
			}
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
	case *MsgGetLiveExecutedServices:
		if msg.VerifyNil {
			if len(a.liveExecutedServices) > 0 {
				break
			}
		}
		if a.pidGetData == nil {
			break
		}
		ctx.Request(a.pidGetData, &MsgGetLiveServiceData{})
	case *MsgLiveServiceData:
		if err := func() error {
			logs.LogBuild.Printf("Get response, GetLiveExecutedServices: %s", msg.Data)
			result := new(Services)
			if err := json.Unmarshal(msg.Data, result); err != nil {
				return err
			}
			a.lastTimestampLiveExecutedServices = result.LastTimestampLiveExecutedServices
			if len(result.LiveExecutedServices) <= 0 {
				return errors.New("error empty LiveExecutedServices result")
			}

			sort.Slice(result.LiveExecutedServices, func(i, j int) bool {
				return result.LiveExecutedServices[j].Timestamp < result.LiveExecutedServices[i].Timestamp
			})

			a.liveExecutedServices = make([]Service, 0)
			for _, v := range result.LiveExecutedServices {
				if (v == Service{}) {
					continue
				}
				if v.TimeService <= v.Timestamp {
					v.TimeService = v.Timestamp + 120*60*1*1000 // 120 minutos
				}
				a.liveExecutedServices = append(a.liveExecutedServices, v)
				if v.Timestamp < time.Now().Add(-24*time.Hour).UnixMilli() {
					break
				}
			}
			return nil
		}(); err != nil {
			logs.LogWarn.Println(err)
			fmt.Printf("GetLiveExecutedServices err: %s\n", err)
			return
		}
		if len(a.liveExecutedServices) > 0 {
			if data, err := json.Marshal(a.liveExecutedServices); err == nil {
				logs.LogInfo.Printf("LiveExecutedServices: %s", data)
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
		if a.liveService.Valid() && ctx.Sender() != nil {
			ctx.Send(ctx.Sender(), &messages.ExternalServiceSch{
				Timestamp:   a.liveService.Timestamp,
				Timeservice: a.liveService.TimeService,
				Itininerary: int32(a.liveService.Itinerary),
				Route:       int32(a.liveService.Ruta),
			})
		}
	case *MsgPublishServices:
		if a.evs != nil {
			fmt.Printf("%s: publish service: %v\n", time.Now().Format("15:04:05.000"), msg.Data)
			fmt.Printf("%s: before service: %v\n", time.Now().Format("15:04:05.000"), a.liveService)
			a.evs.Publish(&messages.ExternalServiceSch{
				Timestamp:   msg.Data.Timestamp,
				Timeservice: msg.Data.TimeService,
				Itininerary: int32(msg.Data.Itinerary),
				Route:       int32(msg.Data.Ruta),
			})
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
	t0_0 := time.NewTimer(1000 * time.Millisecond)
	defer t0_0.Stop()
	t0_1 := time.NewTimer(3000 * time.Millisecond)
	defer t0_1.Stop()
	t1 := time.NewTicker(timeout)
	defer t1.Stop()
	t2 := time.NewTicker(30 * time.Second)
	defer t2.Stop()

	for {
		select {
		case <-t0_0.C:
			rootctx.Send(self, &MsgGetServices{})
		case <-t0_1.C:
			rootctx.Send(self, &MsgTriggerServices{})
		case <-t1.C:
			rootctx.Send(self, &MsgGetServices{})
		case <-t2.C:
			rootctx.Send(self, &MsgTriggerServices{})

		case <-ctx.Done():
			return
		}
	}
}
