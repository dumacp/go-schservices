package main

import (
	"fmt"
	"os"
)

var (
	KeycloakUrl  string
	RedirectUrl  string
	Url          string
	Realm        string
	ClientSecret string
	Clientid     string
	User         string
	Passcode     string
)

func GetENV() {
	if len(KeycloakUrl) <= 0 {
		KeycloakUrl = os.Getenv("KEYCLOAK_URL_DEVICES")
	}
	if len(RedirectUrl) <= 0 {
		RedirectUrl = os.Getenv("REDIRECT_URL_DEVICES")
	}
	if len(Url) <= 0 {
		Url = os.Getenv("URL_SERVICES")
	}
	if len(Realm) <= 0 {
		Realm = os.Getenv("REALM_DEVICES")
	}
	if len(ClientSecret) <= 0 {
		ClientSecret = os.Getenv("CLIENTSECRET_DEVICES")
	}
	if len(Clientid) <= 0 {
		Clientid = os.Getenv("CLIENTID_DEVICES")
	}
	if len(User) <= 0 {
		User = os.Getenv("APPFARE_USER")
	}
	if len(Passcode) <= 0 {
		Passcode = os.Getenv("APPFARE_PASSCODE")
	}

	fmt.Printf("keycloakurl: %s\n", KeycloakUrl)
	fmt.Printf("redirecturl: %s\n", RedirectUrl)
	fmt.Printf("remoteMqttBrokerURL: %s\n", Url)
	fmt.Printf("realm: %s\n", Realm)
	fmt.Printf("clientSecret: %s\n", ClientSecret)
	fmt.Printf("clientid: %s\n", Clientid)
}
