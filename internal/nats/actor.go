package nats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/internal/database"
	"github.com/dumacp/go-schservices/internal/utils"
)

const (
	TIMEOUT            = 30 * time.Second
	SUBJECT_SERVICESCH = "ServicesSchedules"
)

type Actor struct {
	id           string
	propsNats    *actor.Props
	pidNats      *actor.PID
	evs          *eventstream.EventStream
	subscriptors map[string]*eventstream.Subscription
	lastvalue    *gwiotmsg.KvEntryMessage
	cancel       func()
	db           database.DBservice
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

		a.evs = &eventstream.EventStream{}
		a.subscriptors = make(map[string]*eventstream.Subscription)
		conx, cancel := context.WithCancel(context.TODO())
		a.cancel = cancel
		go tick(conx, ctx, TIMEOUT)

		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
	case *actor.Stopping:
		if a.cancel != nil {
			a.cancel()
		}
	case *actor.Terminated:
		if a.pidNats != nil && (msg.GetWho().GetId() == a.pidNats.GetId()) {
			a.pidNats = nil
		}
	case *tickmsg:
		if a.propsNats != nil && a.pidNats == nil {
			pid, err := ctx.SpawnNamed(a.propsNats, "nastio-servicesch")
			if err != nil {
				logs.LogWarn.Printf("spawn natsio-servicesch error: %s", err)
				break
			}
			ctx.Watch(pid)
			a.pidNats = pid

			ctx.Request(a.pidNats, &gwiotmsg.WatchKeyValue{
				Bucket: SUBJECT_SERVICESCH,
				Key:    a.id,
			})
		}
	case *MsgDBdata:
		var backup *gwiotmsg.KvEntryMessage
		data, err := a.db.Get("current", "backup")
		if err != nil {
			logs.LogWarn.Printf("get db data error: %s", err)
			break
		}
		if err := json.Unmarshal(data, backup); err != nil {
			logs.LogWarn.Printf("get db data error: %s", err)
			break
		}
		a.lastvalue = backup
		a.evs.Publish(&MsgRawdata{
			Payload: backup.Data,
		})
	case *gwiotmsg.KvEntryMessage:
		if a.lastvalue != nil && a.lastvalue.Rev >= msg.Rev {
			logs.LogWarn.Printf("same Rev in message: %d", msg.Rev)
			break
		}
		a.lastvalue = msg
		// TODO: select with bucket or key????
		switch msg.Bucket {
		case SUBJECT_SERVICESCH:
			a.lastvalue = msg
			data := make([]byte, len(msg.GetData()))
			copy(data, msg.GetData())
			a.evs.Publish(&MsgRawdata{
				Payload: data,
			})
		}
	case *MsgSubscribe:
		if ctx.Sender() == nil {
			break
		}

		delete(a.subscriptors, ctx.Sender().GetId())
		a.subscriptors[ctx.Sender().GetId()] = subscribe(ctx, a.evs)
		ctx.Respond(&MsgRawdata{
			Payload: a.lastvalue.GetData(),
		})
	}
}

type tickmsg struct{}

func tick(contxt context.Context, ctx actor.Context, timeout time.Duration) {
	initial := time.NewTimer(3 * time.Second)
	defer initial.Stop()

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	ctxroot := ctx.ActorSystem().Root
	self := ctx.Self()

	for {
		select {
		case <-contxt.Done():
			return
		case <-initial.C:
			ctxroot.Send(self, &MsgDBdata{})
			ctxroot.Send(self, &tickmsg{})
		case <-ticker.C:
			ctxroot.Send(self, &tickmsg{})
		}
	}
}
