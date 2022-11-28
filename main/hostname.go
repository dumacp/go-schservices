package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/dumacp/go-logs/pkg/logs"
)

const (
	pathenvfile = "/usr/include/serial-dev"
)

// Hostname get hostname
func Hostname(hostname string) string {
	if len(hostname) > 0 {
		return hostname
	}
	envdev := make(map[string]string)
	if fileenv, err := os.Open(pathenvfile); err != nil {
		logs.LogWarn.Printf("error: reading file env, %s", err)
	} else {
		scanner := bufio.NewScanner(fileenv)
		for scanner.Scan() {
			line := scanner.Text()
			// log.Println(line)
			split := strings.Split(line, "=")
			if len(split) > 1 {
				envdev[split[0]] = split[1]
			}
		}
	}
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		logs.LogError.Fatalf("Error: there is not hostname! %s", err)
	}
	if v, ok := envdev["sn-dev"]; ok {
		reg, err := regexp.Compile("[^a-zA-Z0-9\\-_\\.]+")
		if err != nil {
			log.Println(err)
		}
		processdString := reg.ReplaceAllString(v, "")
		// log.Println(processdString)
		if len(processdString) > 0 {
			hostname = processdString
		}
	}
	return hostname

}
