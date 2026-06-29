package service

import "github.com/linkease/quick-start/istore-backend/models"

var guideSoftSourceIdentities = []string{
	"OpenWrtHttp",
	"OpenWrtHttps",
	"CERNET",
	"SUSTech",
	"Tsinghua",
	"USTC",
	"Alibaba Cloud",
	"Tencent Cloud",
}

var guideSoftSourceNames = []string{
	"OpenWRT(HTTP)",
	"OpenWRT(HTTPS)",
	"高校联合镜像站（智能选择最近大学镜像站）",
	"南方科技大学",
	"清华大学",
	"中国科学技术大学",
	"阿里云",
	"腾讯云",
}

var guideSoftSourceURLs = []string{
	"http://downloads.openwrt.org/",
	"https://downloads.openwrt.org/",
	"https://mirrors.cernet.edu.cn/openwrt/",
	"https://mirrors.sustech.edu.cn/openwrt/",
	"https://mirrors.tuna.tsinghua.edu.cn/openwrt/",
	"https://mirrors.ustc.edu.cn/openwrt/",
	"https://mirrors.aliyun.com/openwrt/",
	"https://mirrors.cloud.tencent.com/openwrt/",
}

func buildGuideSoftSourceByIndex(identity string, index int) models.GuideSoftSourceInfo {
	return models.GuideSoftSourceInfo{
		Identity: identity,
		Name:     guideSoftSourceNames[index],
		URL:      guideSoftSourceURLs[index],
	}
}

func resolveGuideSoftSourceByURL(url string) models.GuideSoftSourceInfo {
	for index, softURL := range guideSoftSourceURLs {
		if softURL == url {
			return models.GuideSoftSourceInfo{
				Identity: guideSoftSourceIdentities[index],
				Name:     guideSoftSourceNames[index],
				URL:      url,
			}
		}
	}
	return models.GuideSoftSourceInfo{
		Identity: url,
		Name:     url,
		URL:      url,
	}
}

func resolveGuideSoftSourceByIdentity(identity string) models.GuideSoftSourceInfo {
	for index, sourceIdentity := range guideSoftSourceIdentities {
		if sourceIdentity == identity {
			return models.GuideSoftSourceInfo{
				Identity: sourceIdentity,
				Name:     guideSoftSourceNames[index],
				URL:      guideSoftSourceURLs[index],
			}
		}
	}
	return models.GuideSoftSourceInfo{}
}
