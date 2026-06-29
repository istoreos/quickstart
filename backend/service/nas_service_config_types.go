package service

import "github.com/linkease/quick-start/istore-backend/modules/nas/serviceconfig"

type NasSambaCreateInput = serviceconfig.SambaCreateInput

type NasWebdavCreateInput = serviceconfig.WebdavCreateInput

type NasServiceStatusReader = serviceconfig.StatusReader

type NasServiceRuntimeReader = serviceconfig.RuntimeReader

type NasServiceConfigWriter = serviceconfig.ConfigWriter

type NasSambaTemplateWriter = serviceconfig.SambaTemplateWriter

func buildNasSambaURL(ipv4addr string, shareName string) string {
	return serviceconfig.BuildSambaURL(ipv4addr, shareName)
}

func buildNasWebdavURL(ipv4addr string, port string) string {
	return serviceconfig.BuildWebdavURL(ipv4addr, port)
}
