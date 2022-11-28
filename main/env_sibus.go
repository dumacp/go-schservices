package main

import "flag"

func init() {
	flag.StringVar(&KeycloakUrl, "keycloakurl", "https://sibus.ambq.gov.co/auth", "keycloak url")
	flag.StringVar(&RedirectUrl, "redirecturl", "https://sibus.ambq.gov.co/", "redirect url")
	flag.StringVar(&Realm, "realm", "DEVICES", "realm by devices")
	flag.StringVar(&Clientid, "clientid", "devices", "client id for devices in keycloak")
	flag.StringVar(&ClientSecret, "clientsecret", "985d96f0-2d5e-487f-8a8a-550cd4616751", "client secrets for client")
	flag.StringVar(&Url, "url", "https://sibus.ambq.gov.co", "url for request services")
	flag.StringVar(&User, "username", "", "username for request services")
	flag.StringVar(&Passcode, "passcode", "", "passcode for request services")
}
