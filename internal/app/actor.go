package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/api/services"
	"github.com/dumacp/go-schservices/internal/constan"
	"github.com/dumacp/go-schservices/internal/nats"
	"github.com/dumacp/go-schservices/internal/utils"
	"github.com/google/uuid"
)

var TIMEOUT = 3 * time.Minute

const (
	DATABASE_PATH = "/SD/bboltdb/servicesdb"
)

type Actor struct {
	id        string
	url       string
	evs       *eventstream.EventStream
	pidData   *actor.PID
	contxt    context.Context
	dataActor actor.Actor
	cancel    func()
}

func NewActor(id, urlrest string, data actor.Actor) actor.Actor {
	if len(id) == 0 {
		id = utils.Hostname()
	}

	a := &Actor{id: id}
	a.url = urlrest
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

		// TODO: verify this remove
		/**
		db, err := database.Open(ctx, DATABASE_PATH)
		if err != nil {
			time.Sleep(3 * time.Second)
			logs.LogError.Panicf("open database (%s) error: %s", DATABASE_PATH, err)
		}

		a.db = db.PID()
		/**/
		if a.dataActor != nil {
			if pidData, err := ctx.SpawnNamed(actor.PropsFromFunc(a.dataActor.Receive), "schsvc-data-actor"); err != nil {
				time.Sleep(3 * time.Second)
				logs.LogError.Panicf("open nats-actor error: %s", err)
			} else {
				a.pidData = pidData
				ctx.Watch(pidData)
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
	case *actor.Terminated:
		logs.LogWarn.Printf("actor terminated (%s)", msg.Who.GetId())
		if a.pidData != nil && (msg.Who.GetId() == a.pidData.GetId()) {
			a.pidData = nil
		}
	case *services.RequestStatusSch:
	case *services.GetCompanyDriverMsg:
		fmt.Printf("get company drivers: %v\n", msg)
		if ctx.Sender() == nil {
			break
		}
		if len(msg.GetCompanyId()) == 0 {
			ctx.Respond(fmt.Errorf("error: company id not found"))
		}
		if len(msg.GetDriverDoc()) == 0 {
			ctx.Respond(fmt.Errorf("error: driver doc not found"))
		}
		if a.pidData == nil {
			ctx.Respond(fmt.Errorf("error: data actor not found"))
			break
		}
		res, err := ctx.RequestFuture(a.pidData, &gwiotmsg.GetKeyValue{
			Key:    msg.GetDriverDoc(),
			Bucket: fmt.Sprintf("%s%s", msg.GetCompanyId(), constan.SUBJECT_SVC_COMPNAY_DRIVER),
		}, 3*time.Second).Result()
		if err != nil {
			ctx.Respond(err)
			break
		}
		switch res := res.(type) {
		case *gwiotmsg.KvEntryMessage:
			// values := make([]*services.ScheduleService, 0)
			val := new(services.Driver)
			if err := json.Unmarshal(res.Data, &val); err != nil {
				fmt.Printf("error unmarshal: %s, data: %s\n", err, res.Data)
				ctx.Respond(err)
				break
			}
			ctx.Respond(&services.CompanyDriverMsg{
				Driver: val,
			})
		default:
			ctx.Respond(fmt.Errorf("error response: %T", res))
		}
	case *services.StartServiceMsg:
		fmt.Printf("start service: %v\n", msg)
		if ctx.Sender() == nil {
			break
		}
		if a.pidData == nil {
			ctx.Respond(fmt.Errorf("error: data actor not found"))
			break
		}
		payload := &StartServicePayload{
			ServiceId: msg.ServiceId,
			DriverId:  msg.DriverId,
			CompanyId: msg.CompanyId,
		}
		uuid, _ := uuid.NewUUID()
		startSvc := &Command{
			DeviceId:   msg.DeviceId,
			PlatformId: msg.PlatformId,
			Type:       "COMMAND",
			Subtype:    "RequestStartLiveExecutedService",
			Payload:    payload,
			MessageId:  uuid.String(),
			Timestamp:  time.Now().UnixMilli(),
		}
		data := startSvc.Bytes()
		if res, err := ctx.RequestFuture(a.pidData, &gwiotmsg.HttpPostRequest{
			Url:  fmt.Sprintf("%s%s", a.url, constan.URL_SVC_COMMAND),
			Data: data,
		}, 10*time.Second).Result(); err != nil {
			ctx.Respond(err)
			break
		} else if resResponse, ok := res.(*gwiotmsg.HttpPostResponse); ok {
			if len(resResponse.Error) > 0 {
				ctx.Respond(&services.StartServiceResponseMsg{Error: resResponse.Error})
			} else if resResponse.Code == 200 {
				resStart := new(CommandResponse)
				if err := json.Unmarshal(resResponse.Data, resStart); err != nil {
					ctx.Respond(&services.StartServiceResponseMsg{
						Error: fmt.Sprintf("error unmarshal: %s, data: %s", err, resResponse.GetData()),
					})
					break
				}
				ctx.Respond(&services.StartServiceResponseMsg{
					DataCode: int32(resStart.Data.Code),
					DataMsg:  resStart.Data.Message,
					Error:    "",
				})
			} else {
				ctx.Respond(&services.StartServiceResponseMsg{
					Error: fmt.Sprintf("error response: %d, %s", resResponse.Code, resResponse.GetData()),
				})
			}
		} else {
			ctx.Respond(&services.StartServiceResponseMsg{
				Error: fmt.Sprintf("error response: %T", res),
			})
		}

	case *services.TakeServiceMsg:
		fmt.Printf("take service: %v\n", msg)
		if ctx.Sender() == nil {
			break
		}
		if a.pidData == nil {
			ctx.Respond(fmt.Errorf("error: data actor not found"))
			break
		}
		payload := &TakeServicePayload{
			ServiceId: msg.ServiceId,
			DriverId:  msg.DriverId,
			CompanyId: msg.CompanyId,
		}
		uuid, _ := uuid.NewUUID()
		takeSvc := &Command{
			DeviceId:   msg.DeviceId,
			PlatformId: msg.PlatformId,
			Type:       "COMMAND",
			Subtype:    "RequestLiveExecutedService",
			Payload:    payload,
			MessageId:  uuid.String(),
			Timestamp:  time.Now().UnixMilli(),
		}
		data := takeSvc.Bytes()
		if res, err := ctx.RequestFuture(a.pidData, &gwiotmsg.HttpPostRequest{
			Url:  fmt.Sprintf("%s%s", a.url, constan.URL_SVC_COMMAND),
			Data: data,
		}, 10*time.Second).Result(); err != nil {
			fmt.Printf("error request 34: %s\n", err)
			ctx.Respond(err)
			break
		} else if resResponse, ok := res.(*gwiotmsg.HttpPostResponse); ok {
			fmt.Printf("error request: %s\n", resResponse)

			if len(resResponse.Error) > 0 {
				// Encuentra y extrae el JSON anidado
				start := strings.Index(resResponse.Error, "resp: {")
				if start != -1 {
					// Encuentra el contenido JSON de 'resp'
					respJSON := resResponse.Error[start+6:] // +6 para omitir "resp: "
					respJSON = strings.TrimSpace(respJSON)
					type RespDetails struct {
						Name string `json:"name"`
						Code int    `json:"code"`
						Msg  string `json:"msg"`
					}
					var details RespDetails
					err = json.Unmarshal([]byte(respJSON), &details)
					if err != nil {
						fmt.Println("Error al deserializar el JSON de 'resp':", err)
						return
					}
					ctx.Respond(&services.TakeServiceResponseMsg{Error: details.Msg})
				} else {
					ctx.Respond(&services.TakeServiceResponseMsg{Error: resResponse.Error})
				}
			} else if resResponse.Code == 200 {
				resTake := new(CommandResponse)
				if err := json.Unmarshal(resResponse.Data, resTake); err != nil {
					ctx.Respond(&services.TakeServiceResponseMsg{
						Error: fmt.Sprintf("error unmarshal: %s, data: %s", err, resResponse.GetData()),
					})
					break
				}
				ctx.Respond(&services.TakeServiceResponseMsg{
					DataCode: int32(resTake.Data.Code),
					DataMsg:  resTake.Data.Message,
					Error:    "",
				})
			} else {
				ctx.Respond(&services.TakeServiceResponseMsg{
					Error: fmt.Sprintf("error response: %d, %s", resResponse.Code, resResponse.GetData()),
				})
			}
		} else {
			ctx.Respond(&services.TakeServiceResponseMsg{
				Error: fmt.Sprintf("error response: %T", res),
			})
		}
	case *services.GetExecutedServiceMsg:
		fmt.Printf("get executed service: %v\n", msg)
		if ctx.Sender() == nil {
			break
		}
		if a.pidData == nil {
			ctx.Respond(fmt.Errorf("error: data actor not found"))
			break
		}
		res, err := ctx.RequestFuture(a.pidData, &gwiotmsg.GetKeyValue{
			Key:            msg.GetDeviceId(),
			Bucket:         constan.SUBJECT_SVC_EXECUTED,
			IncludeHistory: true,
		}, 3*time.Second).Result()
		if err != nil {
			ctx.Respond(err)
			break
		}
		switch res := res.(type) {
		case *gwiotmsg.KvEntryMessage:
			val := new(services.ExecutedService)
			if err := json.Unmarshal(res.Data, val); err != nil {
				fmt.Printf("error unmarshal: %s, data: %s\n", err, res.Data)
				ctx.Respond(err)
				break
			}
			result := make([]*services.ExecutedService, 0)
			ctx.Respond(&services.ExecutedServiceMsg{
				Services: result,
			})
		case *gwiotmsg.KvEntriesMessage:
			result := make([]*services.ExecutedService, 0)
			for _, v := range res.Entries {
				val := new(services.ExecutedService)
				if err := json.Unmarshal(v.Data, val); err != nil {
					fmt.Printf("error unmarshal: %s, data: %s\n", err, v.Data)
					ctx.Respond(err)
					break
				}
				result = append(result, val)
			}
			ctx.Respond(&services.ExecutedServiceMsg{
				Services: result,
			})
		}

	case *services.GetCompanyProgSvcMsg:
		fmt.Printf("get company services: %v\n", msg)
		if ctx.Sender() == nil {
			break
		}
		if a.pidData == nil {
			ctx.Respond(fmt.Errorf("error: data actor not found"))
			break
		}

		if res, err := ctx.RequestFuture(a.pidData, &gwiotmsg.HttpGetRequest{
			Url: fmt.Sprintf("%s%s%s", a.url, constan.URL_SVC_SCHEDULING, msg.GetDeviceId()),
		}, 10*time.Second).Result(); err != nil {
			fmt.Printf("error request 34: %s\n", err)
			ctx.Respond(err)
			break
		} else if resResponse, ok := res.(*gwiotmsg.HttpGetResponse); ok {
			fmt.Printf("error request: %s\n", resResponse)

			if len(resResponse.Error) > 0 {
				// Encuentra y extrae el JSON anidado
				start := strings.Index(resResponse.Error, "resp: {")
				if start != -1 {
					// Encuentra el contenido JSON de 'resp'
					respJSON := resResponse.Error[start+6:] // +6 para omitir "resp: "
					respJSON = strings.TrimSpace(respJSON)
					type RespDetails struct {
						Name string `json:"name"`
						Code int    `json:"code"`
						Msg  string `json:"msg"`
					}
					var details RespDetails
					err = json.Unmarshal([]byte(respJSON), &details)
					if err != nil {
						fmt.Println("Error al deserializar el JSON de 'resp':", err)
						return
					}
					ctx.Respond(&services.CompanyProgSvcMsg{Error: details.Msg})
				} else {
					ctx.Respond(&services.CompanyProgSvcMsg{Error: resResponse.Error})
				}
			} else if resResponse.Code == 200 {
				val := struct {
					Values []*services.ScheduleService `json:"scheduledServices"`
				}{
					Values: make([]*services.ScheduleService, 0),
				}
				if err := json.Unmarshal(resResponse.Data, &val); err != nil {
					fmt.Printf("error unmarshal: %s, data: %s\n", err, resResponse.GetData())
					ctx.Respond(err)
					break
				}

				funcVerify := func(svc *services.ScheduleService) bool {
					switch {
					case msg.GetRouteId() > 0 && len(msg.GetState()) > 0:
						return svc.Itinenary.Id == msg.GetRouteId() && strings.EqualFold(svc.State, msg.State)
					case msg.GetRouteId() > 0:
						return svc.Itinenary.Id == msg.GetRouteId()
					case len(msg.GetState()) > 0:
						return strings.EqualFold(svc.State, msg.State)
					}
					return true
				}
				svcs := make([]*services.ScheduleService, 0)
				for _, v := range val.Values {
					if funcVerify(v) {
						svcs = append(svcs, v)
					}
				}
				ctx.Respond(&services.CompanyProgSvcMsg{
					ScheduledServices: svcs,
				})
			} else {
				ctx.Respond(&services.CompanyProgSvcMsg{
					Error: string(resResponse.GetData()),
				})
			}
		} else {
			ctx.Respond(&services.CompanyProgSvcMsg{
				Error: fmt.Sprintf("error response: %T", res),
			})
		}

		/**
		res, err := ctx.RequestFuture(a.pidData, &gwiotmsg.GetKeyValue{
			Key:    msg.GetCompanyId(),
			Bucket: constan.SUBJECT_SVC_COMPNAY_SNAPSHOT,
		}, 3*time.Second).Result()
		if err != nil {
			ctx.Respond(err)
			break
		}
		switch res := res.(type) {
		case *gwiotmsg.KvEntryMessage:
			// values := make([]*services.ScheduleService, 0)
			val := struct {
				Values []*services.ScheduleService `json:"scheduledServices"`
			}{
				Values: make([]*services.ScheduleService, 0),
			}
			if err := json.Unmarshal(res.Data, &val); err != nil {
				fmt.Printf("error unmarshal: %s, data: %s\n", err, res.Data)
				ctx.Respond(err)
				break
			}

			funcVerify := func(svc *services.ScheduleService) bool {
				switch {
				case msg.GetRouteId() > 0 && len(msg.GetState()) > 0:
					return svc.Itinenary.Id == msg.GetRouteId() && strings.EqualFold(svc.State, msg.State)
				case msg.GetRouteId() > 0:
					return svc.Itinenary.Id == msg.GetRouteId()
				case len(msg.GetState()) > 0:
					return strings.EqualFold(svc.State, msg.State)
				}
				return true
			}
			svcs := make([]*services.ScheduleService, 0)
			for _, v := range val.Values {
				if funcVerify(v) {
					svcs = append(svcs, v)
				}
			}
			ctx.Respond(&services.CompanyProgSvcMsg{
				ScheduledServices: svcs,
			})
		default:
			ctx.Respond(fmt.Errorf("error response: %T", res))
		}
		/**/

	case *services.GetVehProgSvcMsg:
		fmt.Printf("get company driver services: %v\n", msg)
		if ctx.Sender() == nil {
			break
		}
		if a.pidData == nil {
			ctx.Respond(fmt.Errorf("error: data actor not found"))
			break
		}
		res, err := ctx.RequestFuture(a.pidData, &gwiotmsg.GetKeyValue{
			Key:    msg.GetDeviceId(),
			Bucket: constan.SUBJECT_SVC_SNAPSHOT,
		}, 3*time.Second).Result()
		if err != nil {
			ctx.Respond(err)
			break
		}
		switch res := res.(type) {
		case *gwiotmsg.KvEntryMessage:
			// values := make([]*services.ScheduleService, 0)
			val := struct {
				Values []*services.ScheduleService `json:"scheduledServices"`
			}{
				Values: make([]*services.ScheduleService, 0),
			}
			if err := json.Unmarshal(res.Data, &val); err != nil {
				fmt.Printf("error unmarshal: %s, data: %s\n", err, res.Data)
				ctx.Respond(err)
				break
			}

			funcVerify := func(svc *services.ScheduleService) bool {
				// switch {
				// case msg.GetRouteId() > 0 && len(msg.GetState()) > 0:
				// 	return svc.Itinenary.Id == msg.GetRouteId() && strings.EqualFold(svc.State, msg.State)
				// case msg.GetRouteId() > 0:
				// 	return svc.Itinenary.Id == msg.GetRouteId()
				// case len(msg.GetState()) > 0:
				// 	return strings.EqualFold(svc.State, msg.State)
				// }
				return true
			}
			svcs := make([]*services.ScheduleService, 0)
			for _, v := range val.Values {
				if funcVerify(v) {
					svcs = append(svcs, v)
				}
			}
			ctx.Respond(&services.VehProgSvcMsg{
				ScheduledServices: svcs,
			})
		default:
			ctx.Respond(fmt.Errorf("error response: %T", res))
		}

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
			switch update.GetState() {
			default:
				// if update.GetScheduleDateTime() > time.Now().Add(-24*time.Hour).UnixMilli() {
				a.evs.Publish(&services.UpdateServiceMsg{
					Update: update,
				})
				if ctx.Parent() != nil {
					ctx.Request(ctx.Parent(), &services.UpdateServiceMsg{
						Update: update,
					})
				}
				// }
			}
		}

		sa := msg.GetAdditions()
		for _, update := range sa {
			// if update.GetScheduleDateTime() > time.Now().Add(-24*time.Hour).UnixMilli() {
			fmt.Printf("//////////////**************** addition: %v\n", update)
			a.evs.Publish(&services.UpdateServiceMsg{
				Update: update,
			})
			if ctx.Parent() != nil {
				ctx.Request(ctx.Parent(), &services.UpdateServiceMsg{
					Update: update,
				})
			}
			// }
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
					// if update.GetScheduleDateTime() > time.Now().Add(-24*time.Hour).UnixMilli() {
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
				// }
			}()
		}

	case *services.Snapshot:
		ss := msg.GetScheduledServices()
		fmt.Printf("////// services len: %d\n", len(ss))

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
