package restclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/internal/app"
	"golang.org/x/oauth2"
)

const (
	serviceURL      = "%s/api/external-system-gateway/rest/device-service"
	defaultUsername = "dev.nebulae"
	filterHttpQuery = "?deviceId=%s&scheduledServices=%v&liveExecutedServices=%v"
	defaultPassword = "uno.2.tres"
)

const TIMEOUT = 3 * time.Minute

type Actor struct {
	lastGetService  time.Time
	lastGetLive     time.Time
	lastGetSchedule time.Time
	id              string
	userHttp        string
	passHttp        string
	clientid        string
	clientsecret    string
	realm           string
	url             string
	keyUrl          string
	tks             oauth2.TokenSource
	httpClient      *http.Client
}

func NewActor(id, user, pass, url, keyUrl, realm, clientid, clientsecret string) actor.Actor {
	a := &Actor{}
	a.id = id
	a.userHttp = user
	a.passHttp = pass
	a.url = url
	a.keyUrl = keyUrl
	a.clientid = clientid
	a.clientsecret = clientsecret
	a.realm = realm
	return a
}

func (a *Actor) Receive(ctx actor.Context) {
	fmt.Printf("Message arrived in %s: %s, %T, %s\n",
		ctx.Self().GetId(), ctx.Message(), ctx.Message(), ctx.Sender())
	switch ctx.Message().(type) {
	case *actor.Started:
		a.url = fmt.Sprintf(serviceURL, a.url)

		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())

	case *actor.Stopping:

	case *app.MsgGetServiceData:
		if err := func() error {
			if time.Since(a.lastGetService) < 30*time.Second {
				return fmt.Errorf("last GetSchedule was before 30 seconds")
			}
			if a.tks == nil || !(func() bool {
				t, err := a.tks.Token()
				if err != nil {
					return false
				}
				return time.Until(t.Expiry) > 0
			}()) {
				tks, c, err := Token(a.userHttp, a.passHttp, a.keyUrl, a.realm, a.clientid, a.clientsecret)
				if err != nil {
					return err
				}
				a.tks = tks
				a.httpClient = c
			}
			if a.httpClient != nil {
				a.httpClient.CloseIdleConnections()
			}

			resp, _, err := GetData(a.httpClient, a.id, a.url, filterHttpQuery, false, false)
			if err != nil {
				a.httpClient = nil
				return err
			}
			logs.LogBuild.Printf("Get response, GetServices: %s", resp)
			sender := ctx.Sender()
			if sender == nil {
				sender = ctx.Parent()
			}
			if sender != nil {
				data := make([]byte, len(resp))
				copy(data, resp)
				ctx.Respond(&app.MsgServiceData{Data: data})
			}
			a.lastGetService = time.Now()
			return nil

		}(); err != nil {
			logs.LogError.Println(err)
			fmt.Printf("GetServices err: %s\n", err)
			return
		}
	case *app.MsgGetScheduleServiceData:
		if err := func() error {
			if time.Since(a.lastGetSchedule) < 30*time.Second {
				return fmt.Errorf("last GetSchedule was before 30 seconds")
			}
			if a.tks == nil || !(func() bool {
				t, err := a.tks.Token()
				if err != nil {
					return false
				}
				return time.Until(t.Expiry) > 0
			}()) {
				tks, c, err := Token(a.userHttp, a.passHttp, a.keyUrl, a.realm, a.clientid, a.clientsecret)
				if err != nil {
					return err
				}
				a.tks = tks
				a.httpClient = c
			}
			if a.httpClient != nil {
				a.httpClient.CloseIdleConnections()
			}

			resp, _, err := GetData(a.httpClient, a.id, a.url, filterHttpQuery, true, false)
			if err != nil {
				a.httpClient = nil
				return err
			}
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
			return
		}
	case *app.MsgGetLiveServiceData:
		if err := func() error {
			if time.Since(a.lastGetLive) < 30*time.Second {
				return fmt.Errorf("last GetSchedule was before 30 seconds")
			}
			if a.tks == nil || !(func() bool {
				t, err := a.tks.Token()
				if err != nil {
					return false
				}
				return time.Until(t.Expiry) > 0
			}()) {
				// if a.httpClient != nil {
				// 	a.httpClient.CloseIdleConnections()
				// }
				tks, c, err := Token(a.userHttp, a.passHttp, a.keyUrl, a.realm, a.clientid, a.clientsecret)
				if err != nil {
					return err
				}
				a.tks = tks
				a.httpClient = c
			}
			if a.httpClient != nil {
				a.httpClient.CloseIdleConnections()
			}

			resp, _, err := GetData(a.httpClient, a.id, a.url, filterHttpQuery, false, true)
			if err != nil {
				a.httpClient = nil
				return err
			}
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
			return
		}

	}
}

func tick(ctx context.Context, ctxactor actor.Context, timeout time.Duration) {
	rootctx := ctxactor.ActorSystem().Root
	self := ctxactor.Self()
	t0_0 := time.NewTimer(1000 * time.Millisecond)
	defer t0_0.Stop()
	t1 := time.NewTicker(timeout)
	defer t1.Stop()

	for {
		select {
		case <-t0_0.C:
			rootctx.Send(self, &app.MsgGetServiceData{})
		case <-t1.C:
			rootctx.Send(self, &app.MsgGetServiceData{})
		case <-ctx.Done():
			return
		}
	}
}
