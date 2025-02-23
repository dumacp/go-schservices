package app

import "github.com/dumacp/go-schservices/api/services"

type MsgTick struct {
}
type MsgRequestTokenJwt struct{}
type MsgTriggerServices struct{}
type MsgGetServices struct {
	VerifyNil bool
}
type MsgGetScheduledServices struct {
	VerifyNil bool
}
type MsgGetLiveExecutedServices struct {
	VerifyNil bool
}
type MsgService struct {
	Data Service
}
type MsgScheduledServices struct {
	Data map[int64]Service
}
type MsgLiveExecutedServices struct {
	Data map[int64]Service
}
type MsgSubscribeServices struct{}
type MsgPublishServices struct {
	Data *services.ScheduleService
}
type MsgGetInDB struct{}
type MsgKeycloak struct{}
type MsgRequestStatus struct {
}

type MsgGetServiceData struct{}
type MsgServiceData struct {
	Data []byte
}
type MsgGetScheduleServiceData struct{}
type MsgScheduleServiceData struct {
	Data []*services.ScheduleService
}
type MsgGetLiveServiceData struct{}
type MsgLiveServiceData struct {
	Data []byte
}
type MsgStatus struct {
	State bool
}
