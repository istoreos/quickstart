package main

import (
	"os"
	"strings"

	"github.com/linkease/quick-start/istore-backend/lib/logger"
)

var (
	l = logger.DefaultLogger.NewFacility("main", "device logging")
)

func init() {
	l.SetDebug("main", strings.Contains(os.Getenv("STTRACE"), "main") || os.Getenv("STTRACE") == "all")
}
