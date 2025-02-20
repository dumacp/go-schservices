package constan

const (
	SUBJECT_SVC_MODS             = "FMS-SERVICE-SCHEDULES-MODS"
	SUBJECT_SVC_SNAPSHOT         = "FMS-SERVICE-SCHEDULES-SNAPSHOT"
	SUBJECT_SVC_COMPNAY_SNAPSHOT = "FMS-SERVICE-SCHEDULES-COMPANY-SNAPSHOT"
	SUBJECT_SVC_EXECUTED         = "FFMS-SERVICE-EXECUTED"
	SUBJECT_SVC_COMPNAY_DRIVER   = "_FMS-DRIVER"
	URL_SVC_COMMAND              = "/api/external-system-gateway/rest/requestLiveExecutedService"
	URL_NEW_SVC_COMMAND          = "/api/external-system-gateway/rest/requestCreateLiveExecutedService"
	URL_SVC_SCHEDULING           = "/api/external-system-gateway/rest/service-scheduling/"
	URL_SVC_SHIFTS               = "/api/external-system-gateway/rest/service-shitf/"
	URL_SVC_TAKE_SHIFT           = "/api/external-system-gateway/rest/requestServiceSchedulingShift"
	TOPIC_REPLY                  = "schservices/gwiot"
)
