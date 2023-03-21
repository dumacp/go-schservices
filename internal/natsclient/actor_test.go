package nastclient

import (
	"reflect"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

func TestNewActor(t *testing.T) {
	type args struct {
		id       string
		url      string
		keyValue string
		jwtConf  *JwtConf
	}
	tests := []struct {
		name string
		args args
		want actor.Actor
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{
				id:       "TEST-001",
				url:      "nats://127.0.0.1:4222",
				keyValue: "Services",
				jwtConf:  nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctxroot := actor.NewActorSystem().Root

			got := NewActor(tt.args.id, tt.args.url, tt.args.keyValue, tt.args.jwtConf)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewActor() = %v, want %v", got, tt.want)
			}
			propos := actor.PropsFromFunc(got.Receive)
			pid, err := ctxroot.SpawnNamed(propos, "test1")
			if err != nil {
				t.Fatal(err)
			}

			<-time.After(30 * time.Second)
			ctxroot.PoisonFuture(pid).Wait()

		})
	}
}
