package main

import (
	"log"

	"github.com/linkease/quick-start/istore-backend/ubus"
)

func main() {
	ubusSock := "/var/run/ubus.sock"
	sid := "10e407836f595e58d822709418e2b54e"
	client := ubus.NewUbusClient(ubusSock)
	ubusCli, err := ubus.NewUbus(sid, client)
	if err != nil {
		log.Fatal(err)
	}

	err = ubusCli.SessionGet(sid)
	if err != nil {
		log.Fatal(err)
	}
}
