package keyc

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-logs/pkg/logs"
	"golang.org/x/oauth2"
)

type Actor struct {
	id           string
	userHttp     string
	passHttp     string
	clientid     string
	clientsecret string
	realm        string
	url          string
	tks          oauth2.TokenSource
	httpClient   *http.Client
	lockGet      *sync.Mutex
	pidRequest   map[string]*actor.PID
}

func NewActor(id, user, pass, url, realm, clientid, clientsecret string) actor.Actor {
	a := &Actor{}
	a.id = id
	a.userHttp = user
	a.passHttp = pass
	a.url = url
	a.clientid = clientid
	a.clientsecret = clientsecret
	a.realm = realm

	a.pidRequest = make(map[string]*actor.PID)

	return a
}

func (a *Actor) Receive(ctx actor.Context) {
	logs.LogBuild.Printf("Message arrived in servicesSchedActor: %s, %T, %s",
		ctx.Message(), ctx.Message(), ctx.Sender())
	switch ctx.Message().(type) {
	case *actor.Started:
		a.lockGet = &sync.Mutex{}

		logs.LogInfo.Printf("started \"%s\", %v", ctx.Self().GetId(), ctx.Self())

	case *MsgTick:
		if a.httpClient == nil {
			break
		}
		for _, v := range a.pidRequest {
			ctx.Request(v, MsgClient{
				Client: a.httpClient,
			})
		}
	case *MsgGetClient:
		c, err := func() (*http.Client, error) {
			if a.tks == nil || a.httpClient == nil || !(func() bool {
				t, err := a.tks.Token()
				if err != nil {
					return false
				}
				return time.Until(t.Expiry) > 0
			}()) {
				if a.httpClient != nil {
					a.httpClient.CloseIdleConnections()
				}
				fmt.Println(a.userHttp, a.passHttp, a.url, a.realm, a.clientid, a.clientsecret)
				tks, c, err := Token(a.userHttp, a.passHttp, a.url, a.realm, a.clientid, a.clientsecret)
				if err != nil {
					return nil, err
				}
				a.tks = tks
				a.httpClient = c
			}

			return a.httpClient, nil

		}()
		if err != nil {
			logs.LogWarn.Println(err)
			fmt.Printf("keycloak auth err: %s\n", err)
			if ctx.Sender() != nil {
				a.pidRequest[ctx.Sender().GetId()] = ctx.Sender()
			}
		}
		if ctx.Sender() != nil {
			ctx.Respond(&MsgClient{
				Client: c,
			})
		}

	}
}

func tick(ctx context.Context, ctxactor actor.Context, timeout time.Duration) {
	ctxroot := ctxactor.ActorSystem().Root
	self := ctxactor.Self()

	t0 := time.NewTimer(1000 * time.Millisecond)
	defer t0.Stop()

	for {
		select {
		case <-t0.C:
			ctxroot.Send(self, &MsgTick{})
		case <-ctx.Done():
			return
		}
	}
}
