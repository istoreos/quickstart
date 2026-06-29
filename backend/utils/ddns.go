package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

// var ddnsIP1 = "http://members.3322.org/dyndns/getip"
// var ddnsIP1 = "aHR0cDovL21lbWJlcnMuMzMyMi5vcmc="
// myip.ipip.net/s
var ddnsIP1 = "bXlpcC5pcGlwLm5ldA=="

func getDDNSIP2(httpClient *http.Client, ddnsIP string) (string, error) {
	req, err := http.NewRequest("GET", ddnsIP, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("got ip failed")
	}

	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}

	ip := strings.Trim(string(body), " ")
	ip = strings.Trim(ip, "\n")
	ip2 := net.ParseIP(ip)
	if ip2 != nil {
		return ip, nil
	} else {
		return "", errors.New("ip parse failed:" + ip + "$")
	}
}

func GetDDNSIP(httpClient *http.Client) string {
	ipData, _ := base64.StdEncoding.DecodeString(ddnsIP1)
	s1, _ := getDDNSIP2(httpClient, fmt.Sprintf("http://%s/s", string(ipData)))
	//log.Println("ipData=", string(ipData), "s1=", s1, "err=", err)
	return s1
}
