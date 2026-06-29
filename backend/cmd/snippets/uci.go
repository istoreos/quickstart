package main

import (
	"context"
	"fmt"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/service"
	"github.com/linkease/quick-start/istore-backend/utils"
)

func uciMain() {
	devices, ok := uci.GetSections("network", "device")
	fmt.Println("devices=", devices, "ok=", ok)
	if ok {
		ports, ok := uci.Get("network", devices[0], "ports")
		fmt.Println("ports=", ports, "ok=", ok)
	}

	inters, err := service.GetIpInterface()
	fmt.Println("inters=", inters, "err=", err)
	utils.BatchRun(context.Background(), []string{"reboot now > /dev/null 2>&1"}, 0)
}
