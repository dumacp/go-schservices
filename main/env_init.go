package main

import "flag"

func init() {
	flag.StringVar(&KeycloakUrl, "keycloakurl", "", "keycloak url")
	flag.StringVar(&RedirectUrl, "redirecturl", "", "redirect url")
	flag.StringVar(&Realm, "realm", "", "realm by devices")
	flag.StringVar(&Clientid, "clientid", "", "client id for devices in keycloak")
	flag.StringVar(&ClientSecret, "clientsecret", "", "client secrets for client")
	flag.StringVar(&Url, "url", "", "url for request services")
	flag.StringVar(&User, "username", "", "username for request services")
	flag.StringVar(&Passcode, "passcode", "", "passcode for request services")
}
