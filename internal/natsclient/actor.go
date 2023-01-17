package nastclient

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/internal/app"
	"github.com/dumacp/go-schservices/internal/utils"
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
	lastGetService   time.Time
	lastGetLive      time.Time
	lastGetSchedule  time.Time
	id               string
	userHttp         string
	passHttp         string
	clientid         string
	clientsecret     string
	realm            string
	url              string
	keyUrl           string
	conn             *nats.Conn
	cancel           func()
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

func NewActor(id, url string, jwtConf *JwtConf) actor.Actor {
	a := &Actor{}
	a.id = id
	a.url = url
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
	switch ctx.Message().(type) {
	case *actor.Started:

		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())
		a.funcRetryTimeout = utils.Timeout(time.Now().Add(-30*time.Second),
			[]time.Duration{0 * time.Second, 10 * time.Second, 30 * time.Second, 60 * time.Second, 60 * time.Second, 180 * time.Second})
		ctx.Send(ctx.Self(), &MsgStartConn{})
		contxt, cancel := context.WithCancel(context.TODO())
		tick(contxt, ctx, 10*time.Second)
		a.cancel = cancel

	case *actor.Stopping:
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
			return nil
		}(); err != nil {
			logs.LogWarn.Println(err)
			a.conn = nil
			break
		}
		conn, err := NewConn(a.url, opts...)
		if err != nil {
			logs.LogWarn.Println(err)
			a.conn = nil
			break
		}
		a.conn = conn

	case *app.MsgGetServiceData:
		if a.conn != nil {
			break
		}
		if err := func() error {
			if time.Since(a.lastGetService) < 30*time.Second {
				fmt.Println("last GetSchedule was before 30 seconds")
				return nil
			}

			resp := make([]byte, 0)

			jsctx, err := a.conn.JetStream()
			if err != nil {
				return err
			}

			_, err = jsctx.KeyValue("Services")
			if err != nil {
				return err
			}

			logs.LogBuild.Printf("Get response, GetServices: %s", resp)
			if ctx.Sender() != nil {
				data := make([]byte, len(resp))
				copy(data, resp)
				ctx.Respond(&app.MsgServiceData{Data: data})
			}
			a.lastGetService = time.Now()
			return nil

		}(); err != nil {
			logs.LogError.Println(err)
			fmt.Printf("GetServices err: %s\n", err)
			if a.conn != nil {
				a.conn.Close()
			}
			a.conn = nil
			return
		}
	case *app.MsgGetScheduleServiceData:
		if err := func() error {
			if time.Since(a.lastGetSchedule) < 30*time.Second {
				fmt.Println("last GetSchedule was before 30 seconds")
				return nil
			}

			resp := make([]byte, 0)

			logs.LogBuild.Printf("Get response, GetServices: %s", resp)
			if ctx.Sender() != nil {
				data := make([]byte, len(resp))
				copy(data, resp)
				ctx.Respond(&app.MsgScheduleServiceData{Data: data})
			}
			a.lastGetSchedule = time.Now()
			return nil

		}(); err != nil {
			logs.LogError.Println(err)
			fmt.Printf("GetServices err: %s\n", err)
			if a.conn != nil {
				a.conn.Close()
			}
			a.conn = nil
			return
		}
	case *app.MsgGetLiveServiceData:
		if err := func() error {
			if time.Since(a.lastGetLive) < 30*time.Second {
				fmt.Println("last GetSchedule was before 30 seconds")
				return nil
			}
			resp := make([]byte, 0)

			logs.LogBuild.Printf("Get response, GetServices: %s", resp)
			if ctx.Sender() != nil {
				data := make([]byte, len(resp))
				copy(data, resp)
				ctx.Respond(&app.MsgLiveServiceData{Data: data})
			}
			a.lastGetLive = time.Now()
			return nil

		}(); err != nil {
			logs.LogError.Println(err)
			fmt.Printf("GetServices err: %s\n", err)
			if a.conn != nil {
				a.conn.Close()
			}
			a.conn = nil
			return
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
			rootctx.Send(self, &MsgStartConn{})
		case <-ctx.Done():
			return
		}
	}
}
