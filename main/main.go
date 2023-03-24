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
	"github.com/dumacp/go-schservices/internal/pubsub"
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
	finish := make(chan os.Signal, 1)
	signal.Notify(finish, syscall.SIGINT)
	signal.Notify(finish, syscall.SIGTERM)

	for range finish {
		time.Sleep(300 * time.Millisecond)
		log.Print("Finish")
		return
	}
}
