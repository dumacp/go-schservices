package nastclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/internal/app"
	"github.com/dumacp/go-schservices/internal/utils"
	"github.com/dumacp/go-schservices/pkg/messages"
	"github.com/nats-io/nats.go"
)

const (
// serviceURL      = "%s/api/external-system-gateway/rest/device-service"
// defaultUsername = "dev.nebulae"
// filterHttpQuery = "?deviceId=%s&scheduledServices=%v&liveExecutedServices=%v"
// defaultPassword = "uno.2.tres"
)

const TIMEOUT = 3 * time.Minute

type Actor struct {
	id               string
	userHttp         string
	passHttp         string
	clientid         string
	clientsecret     string
	realm            string
	url              string
	keyUrl           string
	keyValue         string
	conn             *nats.Conn
	svcs             *messages.ScheduleServices
	cancel           func()
	cancelKV         func()
	funcRetryTimeout func() bool
}

type JwtConf struct {
	User         string
	Pass         string
	Realm        string
	ClientID     string
	ClientSecret string
	KeycloakURL  string
}

func NewActor(id, url, keyValue string, jwtConf *JwtConf) actor.Actor {
	a := &Actor{}
	a.id = id
	a.url = url
	a.keyValue = keyValue
	if jwtConf != nil {
		a.keyUrl = jwtConf.KeycloakURL
		a.clientid = jwtConf.ClientID
		a.clientsecret = jwtConf.ClientSecret
		a.realm = jwtConf.Realm
		a.userHttp = jwtConf.User
		a.passHttp = jwtConf.Pass
	}
	return a
}

func (a *Actor) Receive(ctx actor.Context) {
	fmt.Printf("Message arrived in %s: %s, %T, %s\n",
		ctx.Self().GetId(), ctx.Message(), ctx.Message(), ctx.Sender())
	switch msg := ctx.Message().(type) {
	case *actor.Started:

		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
		a.funcRetryTimeout = utils.Timeout(time.Now().Add(-30*time.Second),
			[]time.Duration{0 * time.Second, 10 * time.Second, 30 * time.Second, 60 * time.Second, 60 * time.Second, 180 * time.Second})
		ctx.Send(ctx.Self(), &MsgStartConn{})
		contxt, cancel := context.WithCancel(context.TODO())
		go tick(contxt, ctx, 10*time.Second)
		a.cancel = cancel

	case *actor.Stopping:
		if a.cancel != nil {
			a.cancel()
		}
		if a.cancelKV != nil {
			a.cancelKV()
		}
	case *MsgVerifyConn:
		if a.conn != nil || !a.funcRetryTimeout() {
			break
		}
		ctx.Send(ctx.Self(), &MsgStartConn{})
	case *MsgStartConn:
		if a.conn != nil || !a.funcRetryTimeout() {
			break
		}
		opts := make([]nats.Option, 0)
		if err := func() error {
			if len(a.keyUrl) > 0 {
				jwtOpt, err := JwtOpt(a.userHttp, a.passHttp, a.clientid, a.realm, a.clientsecret, a.keyUrl)
				if err != nil {
					return fmt.Errorf("failed attempt connection: %w", err)
				}
				opts = append(opts, jwtOpt)
			}

			conn, err := NewConn(a.url, opts...)
			if err != nil {
				return fmt.Errorf("failed NewConn: %w", err)
			}
			a.conn = conn

			jsctx, err := a.conn.JetStream()
			if err != nil {
				return fmt.Errorf("failed NewConn: %w", err)
			}
			kv, err := jsctx.KeyValue(a.keyValue)
			if err != nil {
				return fmt.Errorf("failed NewConn: %w", err)
			}

			contxt, cancel := context.WithCancel(context.TODO())
			go watch(contxt, ctx, a.id, kv)
			a.cancelKV = cancel

			conn.SetDisconnectErrHandler(func(c *nats.Conn, err error) {
				if !c.IsConnected() {
					logs.LogWarn.Printf("error nats connection: %s", err)
					// TODO: verify if close channel is internal in package
					// cancel()
				}
			})

			return nil

		}(); err != nil {
			logs.LogWarn.Println(err)
			if a.conn != nil {
				a.conn.Close()
			}
			a.conn = nil
			break
		}

	case *app.MsgGetScheduleServiceData:
		if a.svcs != nil && ctx.Sender() != nil {
			svcs := make([]*messages.ScheduleService, 0)
			for i := range a.svcs.GetScheduleServices() {
				svcs = append(svcs, a.svcs.GetScheduleServices()[i])
			}
			ctx.Respond(&app.MsgScheduleServiceData{
				Data: svcs,
			})
		}
	case *MsgServiceData:
		svcs := new(messages.ScheduleServices)
		err := json.Unmarshal(msg.Data, svcs)
		if err != nil {
			logs.LogWarn.Printf("MsgServiceData error (%s): %s", msg.Data, err)
			break
		}
		if a.svcs == nil || svcs.GetVersion() != a.svcs.GetVersion() {
			a.svcs = svcs
			if ctx.Parent() != nil {
				ctx.RequestWithCustomSender(ctx.Self(), svcs, ctx.Parent())
			}
		}
	}
}

func tick(ctx context.Context, ctxactor actor.Context, timeout time.Duration) {
	rootctx := ctxactor.ActorSystem().Root
	self := ctxactor.Self()
	t0 := time.NewTicker(timeout)
	defer t0.Stop()

	for {
		select {
		case <-t0.C:
			rootctx.Send(self, &MsgVerifyConn{})
		case <-ctx.Done():
			return
		}
	}
}

func watch(ctx context.Context, ctxactor actor.Context, key string, ks nats.KeyValue) {
	rootctx := ctxactor.ActorSystem().Root
	self := ctxactor.Self()

	watcher, err := ks.Watch(key)
	if err != nil {
		fmt.Println(err)
		return
	}
	updates := watcher.Updates()
	defer watcher.Stop()

	for {
		select {
		case v, ok := <-updates:
			if !ok {
				fmt.Printf("KV updates closed")
				return
			}

			if v != nil {
				fmt.Printf("data: %v, %v\n", v.Revision(), v.Delta())
				rootctx.Send(self, &MsgServiceData{
					Data: v.Value()})
			}
		case <-ctx.Done():
			return
		}
	}
}
