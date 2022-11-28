package app

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

var t = time.Now()

func DataTest(ctx actor.Context) {
	switch ctx.Message().(type) {
	case *MsgGetServiceData:
		tl := t.UnixMilli()
		services := Services{
			DeviceID:                          "NE-RC9-2957",
			LastTimestampScheduledServices:    tl,
			LastTimestampLiveExecutedServices: tl,
		}
		data_services, err := json.Marshal(&services)
		if err != nil {
			fmt.Println(err)
			break
		}
		data := []byte(data_services)
		if ctx.Sender() != nil {
			ctx.Respond(&MsgServiceData{
				Data: data,
			})
		}
	case *MsgGetLiveServiceData:
		tl := t.Add(1 * time.Minute).UnixMilli()
		services := Services{
			DeviceID: "NE-RC9-2957",
			LiveExecutedServices: []Service{
				{
					Timestamp:   tl,
					TimeService: tl + 2*60*60*1000,
					Ruta:        23,
					Itinerary:   26,
				},
			},
			LastTimestampScheduledServices:    0,
			LastTimestampLiveExecutedServices: tl,
		}
		data_services, err := json.Marshal(&services)
		if err != nil {
			fmt.Println(err)
			break
		}
		data := []byte(data_services)
		if ctx.Sender() != nil {
			ctx.Respond(&MsgLiveServiceData{
				Data: data,
			})
		}
	case *MsgGetScheduleServiceData:
		tl := t.Add(1 * time.Minute).UnixMilli()
		services := Services{
			DeviceID: "NE-RC9-2957",
			ScheduledServices: []Service{
				{
					Timestamp:   tl,
					TimeService: tl + 2*60*60*1000,
					Ruta:        23,
					Itinerary:   25,
				},
			},
			LastTimestampScheduledServices:    tl,
			LastTimestampLiveExecutedServices: tl,
		}
		data_services, err := json.Marshal(&services)
		if err != nil {
			fmt.Println(err)
			break
		}
		data := []byte(data_services)
		if ctx.Sender() != nil {
			ctx.Respond(&MsgScheduleServiceData{
				Data: data,
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
				time.Sleep(30 * time.Second)
			}
		})
	}
}
