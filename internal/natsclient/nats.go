package nastclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"golang.org/x/oauth2"
)

func JwtOpt(user, pass, clientid, realm, clientsecret, keycloakUrl string) (nats.Option, error) {
	ctx_ := context.TODO()

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	cl := &http.Client{
		Transport: &http.Transport{
			Dial:                  (dialer).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	ctx := context.WithValue(ctx_, oauth2.HTTPClient, cl)

	fmt.Printf("client 1: %v\n", cl)

	// tks, err := TokenSource(ctx, "dumawork", "dumawork", keycloakUrl, "DEVICES", "devices-nats", "f207031c-8384-432a-9ea5-ce9be8121712")
	tks, err := TokenSource(ctx, user, pass, keycloakUrl, realm, clientid, clientsecret)
	if err != nil {
		return nil, err
	}

	tk, err := tks.Token()
	if err != nil {
		return nil, err
	}

	fmt.Printf("token: %s\n", tk.AccessToken)

	jwtOpt := nats.SetJwtBearer(func() string {
		t, _ := tks.Token()
		return t.AccessToken
	})

	return jwtOpt, nil

}

func NewConn(url string, opts ...nats.Option) (*nats.Conn, error) {

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
