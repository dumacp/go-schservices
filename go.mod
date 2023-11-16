module github.com/dumacp/go-schservices

go 1.19

replace github.com/nats-io/nats.go => ../../nats-io/nats.go

replace github.com/dumacp/go-gwiot => ../go-gwiot

require (
	github.com/coreos/go-oidc v2.2.1+incompatible
	github.com/dumacp/go-actors v0.0.0-20230714144318-a7af7e9e701a
	github.com/dumacp/go-gwiot v0.0.0-00010101000000-000000000000
	github.com/dumacp/go-logs v0.0.1
	github.com/eclipse/paho.mqtt.golang v1.4.2
	golang.org/x/oauth2 v0.6.0
)

require (
	github.com/Workiva/go-datastructures v1.0.53 // indirect
	github.com/asynkron/gofun v0.0.0-20220329210725-34fed760f4c2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/lithammer/shortuuid/v4 v4.0.0 // indirect
	github.com/looplab/fsm v1.0.1 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/orcaman/concurrent-map v1.0.0 // indirect
	github.com/prometheus/client_golang v1.16.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/twmb/murmur3 v1.1.6 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk v1.16.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/exp v0.0.0-20221012134508-3640c57a48ea // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sync v0.2.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20230306155012-7f2fa6fef1f4 // indirect
	google.golang.org/grpc v1.55.0 // indirect
)

require (
	github.com/asynkron/protoactor-go v0.0.0-20230414121700-22ab527f4f7a
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	google.golang.org/protobuf v1.31.0
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
)

replace github.com/asynkron/protoactor-go => ../../asynkron/protoactor-go
