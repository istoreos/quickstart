package service

import (
	"log"
	"testing"
)

func TestGetDistFeedUrl(t *testing.T) {
	content := `src/gz istoreos_core https://mirrors.tuna.tsinghua.edu.cn/openwrt/releases/21.02.3/targets/x86/64/packages
src/gz istoreos_base https://mirrors.tuna.tsinghua.edu.cn/openwrt/releases/21.02.3/packages/x86_64/base
src/gz istoreos_luci https://mirrors.tuna.tsinghua.edu.cn/openwrt/releases/21.02.3/packages/x86_64/luci
src/gz istoreos_packages https://mirrors.tuna.tsinghua.edu.cn/openwrt/releases/21.02.3/packages/x86_64/packages
src/gz istoreos_routing https://mirrors.tuna.tsinghua.edu.cn/openwrt/releases/21.02.3/packages/x86_64/routing
src/gz istoreos_telephony https://mirrors.tuna.tsinghua.edu.cn/openwrt/releases/21.02.3/packages/x86_64/telephony
	`
	urlStr, err := getDistFeedUrlByContent(content)
	if err != nil {
		t.Error(err)
	}
	log.Println("urlStr=", urlStr)
	if urlStr != "https://mirrors.tuna.tsinghua.edu.cn/openwrt/" {
		t.Error("not equal")
	}
	content = `
src/gz openwrt_base https://downloads.openwrt.org/releases/21.02.3/packages/aarch64_cortex-a53/base
src/gz openwrt_luci https://downloads.openwrt.org/releases/21.02.3/packages/aarch64_cortex-a53/luci
src/gz openwrt_packages https://downloads.openwrt.org/releases/21.02.3/packages/aarch64_cortex-a53/packages
	`
	urlStr, err = getDistFeedUrlByContent(content)
	if err != nil {
		t.Error(err)
	}
	log.Println("urlStr=", urlStr)
	if urlStr != "https://downloads.openwrt.org/" {
		t.Error("not equal")
	}
}
