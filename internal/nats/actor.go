package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
	"github.com/asynkron/protoactor-go/remote"
	"github.com/dumacp/go-actors/database"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg/gwiot"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/api/services"
	"github.com/dumacp/go-schservices/internal/constan"
	"github.com/dumacp/go-schservices/internal/messages"
	"github.com/dumacp/go-schservices/internal/utils"
)

const (
	TIMEOUT = 30 * time.Second
)

type Actor struct {
	id             string
	remoteAddress  string
	actorDiscovery actor.Actor
	pidNats        *actor.PID
	pidDiscovery   *actor.PID
	evs            *eventstream.EventStream
	subscriptors   map[string]*eventstream.Subscription
	lastvalue      *gwiotmsg.KvEntryMessage
	cancel         func()
	db             database.DBservice
}

func NewActor(id string, actorDiscovery actor.Actor) actor.Actor {
	if len(id) == 0 {
		id = utils.Hostname()
	}
	a := &Actor{id: id}
	a.actorDiscovery = actorDiscovery
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
	logs.LogBuild.Printf("Message arrived in %s: %s, %T, %s",
		ctx.Self().GetId(), ctx.Message(), ctx.Message(), ctx.Sender())
	switch msg := ctx.Message().(type) {
	case *actor.Started:

		if a.actorDiscovery != nil {
			if pid, err := ctx.SpawnNamed(actor.PropsFromFunc(a.actorDiscovery.Receive), "discover-actor"); err != nil {
				time.Sleep(3 * time.Second)
				logs.LogError.Panicf("spawn discover actor error: %s", err)
			} else {
				a.pidDiscovery = pid
			}
		}
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
		if a.pidNats != nil {
			ctx.PoisonFuture(a.pidNats).Wait()
		}
	case *actor.Terminated:
		if a.pidNats != nil && (msg.GetWho().GetId() == a.pidNats.GetId()) {
			a.pidNats = nil
		}
	case *tickmsg:
		if len(a.remoteAddress) <= 0 {
			if a.pidDiscovery != nil {
				disc := &gwiotmsg.Discovery{
					Reply: constan.TOPIC_REPLY,
				}
				ctx.Request(a.pidDiscovery, disc)
			}
		} else if a.pidNats == nil {
			r := remote.GetRemote(ctx.ActorSystem())
			pidResponse, err := r.SpawnNamed(a.remoteAddress, "nast-schsvc-client", gwiot.KIND_NAME, 3*time.Second)
			if err != nil {
				logs.LogWarn.Printf("remote activation nast error: %s", err)
				a.remoteAddress = ""
				break
			}

			pid := pidResponse.GetPid()
			ctx.Watch(pid)
			a.pidNats = pid

			ctx.Request(a.pidNats, &gwiotmsg.WatchKeyValue{
				Bucket: constan.SUBJECT_SVC_SNAPSHOT,
				Key:    a.id,
			})

			go func() {
				time.Sleep(6 * time.Second)
				ctx.Request(a.pidNats, &gwiotmsg.WatchKeyValue{
					Bucket: constan.SUBJECT_SVC_MODS,
					Key:    a.id,
				})
			}()
		}

	case *gwiotmsg.DiscoveryResponse:
		a.remoteAddress = fmt.Sprintf("%s:%d", msg.GetHost(), msg.GetPort())
		ctx.Send(ctx.Self(), &tickmsg{})
	case *MsgDBdata:
		var backup *gwiotmsg.KvEntryMessage
		if a.db == nil {
			logs.LogWarn.Printf("databse is nil")
			break
		}
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
		// a.evs.Publish(&MsgRawdata{
		// 	Payload: backup.Data,
		// })
	case *gwiotmsg.WatchMessage:
		mss := msg.GetKvEntryMessage()
		if a.lastvalue != nil && a.lastvalue.Rev >= mss.Rev {
			logs.LogWarn.Printf("same Rev in message: %d", mss.Rev)
			break
		}
		a.lastvalue = mss
		// TODO: select with bucket or key????
		switch mss.Bucket {
		case constan.SUBJECT_SVC_SNAPSHOT:
			a.lastvalue = mss
			data := make([]byte, len(mss.GetData()))
			copy(data, mss.GetData())
			snap := new(services.Snapshot)
			if err := json.Unmarshal(data, snap); err != nil {
				logs.LogWarn.Printf("error parse message: %s", err)
				break
			}
			a.evs.Publish(snap)
			if ctx.Parent() != nil {
				ctx.Send(ctx.Parent(), snap)
			}

		case constan.SUBJECT_SVC_MODS:
			a.lastvalue = mss
			data := make([]byte, len(mss.GetData()))
			copy(data, mss.GetData())
			snap := new(services.Mods)
			if err := json.Unmarshal(data, snap); err != nil {
				logs.LogWarn.Printf("error parse message: %s", err)
				break
			}
			a.evs.Publish(snap)
			if ctx.Parent() != nil {
				ctx.Send(ctx.Parent(), snap)
			}
		}

	case *MsgSubscribe:
		if ctx.Sender() == nil {
			break
		}

		delete(a.subscriptors, ctx.Sender().GetId())
		a.subscriptors[ctx.Sender().GetId()] = subscribe(ctx, a.evs)
		ctx.Respond(&messages.MsgRawdata{
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
