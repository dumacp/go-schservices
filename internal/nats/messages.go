package nats

type MsgDBdata struct{}
type MsgRawdata struct {
	Payload []byte
}
type MsgSubscribe struct{}

type MsgRequeststatus struct {
}
type MsgStatus struct {
	State bool
}
