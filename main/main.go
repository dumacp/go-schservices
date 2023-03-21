package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/remote"
	"github.com/dumacp/go-logs/pkg/logs"
	"github.com/dumacp/go-schservices/internal/app"
	"github.com/dumacp/go-schservices/internal/pubsub"

	"github.com/dumacp/go-schservices/internal/restclient"
	"github.com/dumacp/go-schservices/pkg/schservices"
)

const (
	NAME_INSTANCE          = "schservices"
	NAME_INSTANCE_KEYCLOAK = "sch-keycloak"
)

const (
	port    = 8090
	version = "1.1.0"
)

var (
	id          string
	debug       bool
	logStd      bool
	showversion bool
)

func init() {
	flag.StringVar(&id, "id", "", "override id for device")
	flag.BoolVar(&debug, "debug", false, "debug")
	flag.BoolVar(&logStd, "logStd", false, "send logs to stdout")
	flag.BoolVar(&showversion, "version", false, "show version")
}

func main() {

	flag.Parse()
	if showversion {
		fmt.Printf("version: %s\n", version)
		os.Exit(2)
	}

	GetENV()

	sys := actor.NewActorSystem()

	pubsub.Init(sys.Root)

	portlocal := portlocal()

	rconfig := remote.Configure("127.0.0.1", portlocal)
	r := remote.NewRemote(sys, rconfig)
	r.Start()

	id = Hostname(id)

	if len(User) <= 0 && len(Passcode) <= 0 {
		Passcode = id
	}
	if len(User) <= 0 {
		User = id
	}

	dataActor := restclient.NewActor(id, User, Passcode, Url, KeycloakUrl, Realm, Clientid, ClientSecret)
	dataProps := actor.PropsFromFunc(dataActor.Receive)

	appactor, err := app.NewActor(id, dataProps)
	if err != nil {
		logs.LogError.Fatalln(err)
	}
	appProps := actor.PropsFromFunc(appactor.Receive)
	appPid, err := sys.Root.SpawnNamed(appProps, NAME_INSTANCE)
	if err != nil {
		logs.LogError.Fatalln(err)
	}

	if err := pubsub.Subscribe(schservices.DISCV_TOPIC, appPid, app.Discover); err != nil {
		logs.LogError.Fatalln(err)
	}

	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)

	for range finish {
		sys.Root.Poison(appPid)
		time.Sleep(300 * time.Millisecond)
		log.Print("Finish")
		return
	}
}
