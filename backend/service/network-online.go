package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/linkease/quick-start/istore-backend/utils"
)

const (
	checkOKSec     = 90
	checkFailedSec = 5
	dialTimeout    = 3
)

type NetworkOnlineStatus int

const (
	NetworkOnlineDetech NetworkOnlineStatus = iota
	NetworkOnlineFailedDns
	NetworkOnlineFailedOffline
	NetworkOnlineFailedSoftSource
	NetworkOnlineOK
)

func (status NetworkOnlineStatus) String() string {
	switch status {
	case NetworkOnlineDetech:
		return "netDetecting"
	case NetworkOnlineFailedDns:
		return "dnsFailed"
	case NetworkOnlineFailedOffline:
		return "netFailed"
	case NetworkOnlineFailedSoftSource:
		return "softSourceFailed"
	case NetworkOnlineOK:
		return "netSuccess"
	}
	return "unknown"
}

type NetworkOnlineChecker struct {
	mu        sync.Mutex
	lastCheck time.Time
	checking  bool
	status    NetworkOnlineStatus
	cacheKey  string
}

func NewNetworkOnlineChecker() *NetworkOnlineChecker {
	return &NetworkOnlineChecker{
		status: NetworkOnlineDetech,
	}
}

func (checker *NetworkOnlineChecker) GetStatus(ip string, gateway string, dns []string) NetworkOnlineStatus {
	var lastStatus NetworkOnlineStatus
	var checkSec time.Duration
	var needCheck bool
	if dns == nil {
		dns = []string{}
	}
	feedStr, feedErr := getDistFeedUrlForCheck()
	if feedErr != nil {
		feedStr = ""
	}
	cacheKey := ip + "|" + gateway + "|" + strings.Join(dns, ",") + "|" + feedStr
	now := time.Now()
	checker.mu.Lock()
	if checker.checking {
		needCheck = false
	} else {
		lastStatus = checker.status
		if lastStatus == NetworkOnlineOK {
			checkSec = checkOKSec
		} else {
			checkSec = checkFailedSec
		}
		if checker.cacheKey == cacheKey && checker.lastCheck.After(now.Add(time.Second*(0-checkSec))) {
			// not timeout, use the old value
			needCheck = false
		} else {
			// detech again
			needCheck = true
			checker.checking = true
			if lastStatus != NetworkOnlineOK {
				// Reset to detecting
				checker.status = NetworkOnlineDetech
				lastStatus = NetworkOnlineDetech
			}
		}
	}
	checker.mu.Unlock()

	if needCheck {
		go checker.doCheck(cacheKey, feedStr)
	}

	return lastStatus
}

func closeConn(c net.Conn) {
	if c != nil {
		c.Close()
	}
}

func getDistFeedUrlForCheck() (string, error) {
	buf, err := ioutil.ReadFile("/etc/opkg/distfeeds.conf")
	if err != nil {
		return "", err
	}
	found := matchStringOnce(string(buf), `(?m)_base\s+(https?:\/\/.*[^\/])\/?$`)
	if found == nil {
		return "", errors.New("feed not found")
	}
	return found[1] + "/Packages.sig", nil
}

func (checker *NetworkOnlineChecker) doCheck(cacheKey string, feedCheckURL string) {
	var status NetworkOnlineStatus
	var c net.Conn
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*dialTimeout)
	defer cancel()
	status = NetworkOnlineFailedSoftSource

	resolver := net.DefaultResolver
	// check DNS ok ?
	_, err = resolver.LookupIP(ctx, "ip", "www.baidu.com")
	if err != nil {
		// check again
		_, err = resolver.LookupIP(ctx, "ip", "www.qq.com")
		if err != nil {
			status = NetworkOnlineFailedDns
		}
	}

	// check network ok ?
	if status == NetworkOnlineFailedDns {
		// dns failed, check network again
		c, err = net.DialTimeout("tcp", "114.114.114.114:53", time.Second*dialTimeout)
		closeConn(c)
		if err != nil {
			// check again
			c, err = net.DialTimeout("tcp", "223.5.5.5:53", time.Second*dialTimeout)
			closeConn(c)
		}
		if err != nil {
			status = NetworkOnlineFailedOffline
		}
	}

	// check softSource ok ?
	if status == NetworkOnlineFailedSoftSource {
		if feedCheckURL != "" {
			l.Debugf("Checking feed URL: %v", feedCheckURL)
			cmdStr := fmt.Sprintf("curl --fail --show-error --max-time 5 -o /dev/null -s -L '%v'", feedCheckURL)
			_, err := utils.BatchOutputCmd(context.Background(), cmdStr, 0)
			if err == nil {
				status = NetworkOnlineOK
			}
		} else {
			// If distFeedUrl not found in other OpenWRT distribute, mark network OK
			status = NetworkOnlineOK
		}
	}

	checker.mu.Lock()
	checker.lastCheck = time.Now()
	checker.cacheKey = cacheKey
	checker.status = status
	checker.checking = false
	checker.mu.Unlock()
}

func (checker *NetworkOnlineChecker) Reset() {
	checker.mu.Lock()
	defer checker.mu.Unlock()
	checker.lastCheck = time.Time{}
}

type ForeignChecker struct {
	mu          sync.Mutex
	dialer      *net.Dialer
	lastCheck   time.Time
	checking    bool
	status      NetworkOnlineStatus
	publicIPv4  string
	countDevice int
}

func NewForeignChecker() *ForeignChecker {
	return &ForeignChecker{
		status: NetworkOnlineDetech,
		dialer: &net.Dialer{
			Timeout: 5 * time.Second,
		},
	}
}

func (checker *ForeignChecker) GetStatus(client *http.Client) (NetworkOnlineStatus, string, int) {
	var lastStatus NetworkOnlineStatus
	var ip string
	var devices int
	var checkSec time.Duration
	var needCheck bool
	now := time.Now()
	checker.mu.Lock()
	ip = checker.publicIPv4
	devices = checker.countDevice
	if checker.checking {
		needCheck = false
	} else {
		lastStatus = checker.status
		if lastStatus == NetworkOnlineOK {
			checkSec = checkOKSec
		} else {
			checkSec = checkFailedSec
		}
		if checker.lastCheck.After(now.Add(time.Second * (0 - checkSec))) {
			needCheck = false
		} else {
			needCheck = true
			checker.checking = true
			if lastStatus != NetworkOnlineOK {
				checker.status = NetworkOnlineDetech
				lastStatus = NetworkOnlineDetech
			}
		}
	}
	checker.mu.Unlock()

	if needCheck {
		go checker.doCheck(client)
	}

	return lastStatus, ip, devices
}

func (checker *ForeignChecker) doCheck(client *http.Client) {
	status := NetworkOnlineFailedOffline
	c, err := tls.DialWithDialer(checker.dialer, "tcp", "www.google.com:443", nil)
	if c != nil {
		c.Close()
	}
	if err == nil {
		status = NetworkOnlineOK
	}
	ip := utils.GetDDNSIP(client)
	var countDevice int
	resp, err := NetworkDeviceList(context.TODO())
	if err == nil {
		countDevice = len(resp.Result.Devices)
	}
	checker.mu.Lock()
	checker.lastCheck = time.Now()
	checker.publicIPv4 = ip
	checker.countDevice = countDevice
	checker.status = status
	checker.checking = false
	checker.mu.Unlock()
}
