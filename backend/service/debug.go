package service

import (
	"os"
	"strings"

	"github.com/linkease/quick-start/istore-backend/lib/logger"
)

var (
	l = logger.DefaultLogger.NewFacility("service", "service logging")
)

func init() {
	l.SetDebug("service", strings.Contains(os.Getenv("STTRACE"), "service") || os.Getenv("STTRACE") == "all")
}
