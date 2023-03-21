package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-schservices/pkg/messages"
)

func DataTest(ctx actor.Context) {
	fmt.Printf("message in test: %s, %T\n", ctx.Message(), ctx.Message())
	switch ctx.Message().(type) {
	case *actor.Started:
		if ctx.Parent() != nil {
			ctx.RequestWithCustomSender(ctx.Self(), &MsgGetScheduleServiceData{}, ctx.Parent())
		}
	case *MsgGetScheduleServiceData:
		services := []*messages.ScheduleService{
			{
				Id:               "1",
				State:            messages.State_READY,
				OrganizationId:   "1-2-3",
				ScheduleDateTime: time.Now().Add(-10 * time.Second).UnixMilli(),
				Itinerary: &messages.Itinerary{
					Id:   "1-1",
					Name: "ITI-1-1",
				},
				Route: &messages.Route{
					Id:   "1",
					Code: "111a",
					Name: "RUTA-1",
				},
				Driver: &messages.Driver{
					Id:       "000",
					Fullname: "test test",
				},
				DriverIds: []string{"000", "111", "222"},
			},
			{
				Id:               "2",
				State:            messages.State_READY,
				OrganizationId:   "1-2-3",
				ScheduleDateTime: time.Now().Add(-40 * time.Minute).UnixMilli(),
				Itinerary: &messages.Itinerary{
					Id:   "1-1",
					Name: "ITI-1-2",
				},
				Route: &messages.Route{
					Id:   "1",
					Code: "111a",
					Name: "RUTA-1",
				},
				Driver: &messages.Driver{
					Id:       "000",
					Fullname: "test test",
				},
				DriverIds: []string{"000", "111", "222"},
			},
			{
				Id:               "3",
				State:            messages.State_READY,
				OrganizationId:   "1-2-3",
				ScheduleDateTime: time.Now().Add(30 * time.Second).UnixMilli(),
				Itinerary: &messages.Itinerary{
					Id:   "1-1",
					Name: "ITI-1-3",
				},
				Route: &messages.Route{
					Id:   "1",
					Code: "111a",
					Name: "RUTA-1",
				},
				Driver: &messages.Driver{
					Id:       "000",
					Fullname: "test test",
				},
				DriverIds: []string{"000", "111", "222"},
			},
		}
		if ctx.Sender() != nil {
			ctx.Respond(&MsgScheduleServiceData{
				Data: services,
			})
		}
	}
}

func TestNewActor(t *testing.T) {

	TIMEOUT = 35 * time.Second
	sys := actor.NewActorSystem()
	rootctx := sys.Root
	propsData := actor.PropsFromFunc(DataTest)
	paramsActor, err := NewActor("FLO-W7-2957", propsData)
	if err != nil {
		t.Fatal(err)
	}
	props := actor.PropsFromFunc(paramsActor.Receive)
	pid, err := rootctx.SpawnNamed(props, "sch-actor")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)

	type args struct {
		pid      *actor.PID
		rootctx  *actor.RootContext
		messages []interface{}
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test1",
			args: args{
				rootctx:  rootctx,
				pid:      pid,
				messages: []interface{}{&MsgTriggerServices{}, &MsgTriggerServices{}, &MsgTriggerServices{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, msg := range tt.args.messages {
				rootctx.RequestFuture(tt.args.pid, msg, time.Millisecond*900).Result()
				time.Sleep(20 * time.Second)
			}
		})
	}
}
