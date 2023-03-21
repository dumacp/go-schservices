package nastclient

type MsgStartConn struct{}
type MsgVerifyConn struct{}
type MsgServiceData struct {
	Data []byte
}
