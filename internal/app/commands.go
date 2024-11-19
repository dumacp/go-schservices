package app

import (
	"encoding/json"
)

type Command struct {
	DeviceId   string      `json:"deviceId"`
	PlatformId string      `json:"platformId"`
	Type       string      `json:"type"`
	Subtype    string      `json:"subtype"`
	From       string      `json:"from"`
	To         string      `json:"to"`
	Payload    interface{} `json:"payload"`
	MessageId  string      `json:"messageId"`
	Timestamp  int64       `json:"timestamp"`
}

func (svc *Command) Bytes() []byte {
	jsonValue, _ := json.Marshal(svc)
	return jsonValue
}

type CommandResponse struct {
	Data   CommandResponseData   `json:"data"`
	Result CommandResponseResult `json:"result"`
}

type CommandResponseData struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type CommandResponseResult struct {
	Code int `json:"code"`
}

type TakeServicePayload struct {
	ServiceId string `json:"serviceId"`
	DriverId  string `json:"driverId"`
	CompanyId string `json:"companyId"`
}

type StartServicePayload struct {
	ServiceId string `json:"serviceId"`
	DriverId  string `json:"driverId"`
	CompanyId string `json:"companyId"`
}
