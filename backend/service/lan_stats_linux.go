//go:build linux

package service

import (
	"net/netip"
	"time"

	"github.com/ti-mo/conntrack"
)

const (
	testIP          = "192.168.9.3"
	lanStatTimeTick = 3 * time.Second
)

type LanStats struct {
	tm        *time.Timer
	c         *conntrack.Conn
	isRunning bool
	// Auto stop when no request after 30s
	lastRunningTime  time.Time
	hosts            map[string]*LanStatHost
	fetchHostStatsCh chan *fetchHostStatsReq
}

type LanStatHost struct {
	ip       string
	rxTotal  int64
	txTotal  int64
	lastItem *NetworkStatisticsItem
	items    []*NetworkStatisticsItem
}

type fetchHostStatsReq struct {
	hostIP    string
	speedOnly bool
	itemCh    chan []*LanHostRet
}

func NewLanStats() *LanStats {
	stats := &LanStats{
		hosts:            make(map[string]*LanStatHost),
		fetchHostStatsCh: make(chan *fetchHostStatsReq, 1),
	}
	go stats.run()
	return stats
}

func (lstat *LanStats) run() {
	if lstat.tm != nil {
		return
	}
	lstat.tm = time.NewTimer(lanStatTimeTick)
	defer lstat.tm.Stop()
	for {
		if err := lstat.runOnce(lstat.tm); err != nil {
			return
		}
	}
}

func (lstat *LanStats) runOnce(tm *time.Timer) error {
	select {
	case req := <-lstat.fetchHostStatsCh:
		if lstat.isRunning {
			lstat.lastRunningTime = time.Now()
			// Return the olds
			lstat.respHosts(req)
			return nil
		}
		if !tm.Stop() {
			select {
			case <-tm.C:
			default:
			}
		}
		tm.Reset(lanStatTimeTick)
		lstat.lastRunningTime = time.Now()
		lstat.isRunning = true
		err := lstat.fetchItems()
		if err != nil {
			l.Debugln("fetchItems in fetchStatCh err=", err)
		}

		// Return the news
		lstat.respHosts(req)

	case <-tm.C:
		now := time.Now()
		if lstat.lastRunningTime.Add(10 * lanStatTimeTick).Before(now) {
			lstat.isRunning = false
			tm.Stop()
			lstat.closeConn()
			return nil
		}
		err := lstat.fetchItems()
		if err != nil {
			l.Debugln("fetchItems in timer err=", err)
		}
		tm.Reset(lanStatTimeTick)
	}
	return nil
}

func (lstat *LanStats) dial() error {
	if lstat.c != nil {
		return nil
	}
	c, err := conntrack.Dial(nil)
	if err != nil {
		l.Debugln("dial to conntrack failed, err=", err)
		return err
	}
	lstat.c = c
	return nil
}

func (lstat *LanStats) closeConn() {
	if lstat.c != nil {
		lstat.c.Close()
		lstat.c = nil
	}
}

func (lstat *LanStats) fetchItems() error {
	err := lstat.dial()
	if err != nil {
		return err
	}
	c := lstat.c
	flows, err := c.Dump(nil)
	if err != nil {
		return err
	}

	for _, flow := range flows {
		// Get source and destination IPs
		srcIP := flow.TupleOrig.IP.SourceAddress
		dstIP := flow.TupleOrig.IP.DestinationAddress

		// Skip broadcast IPs (like 255.255.255.255)
		if srcIP.IsUnspecified() || dstIP.IsUnspecified() ||
			srcIP.IsMulticast() || dstIP.IsMulticast() {
			continue
		}

		// Get upstream (outbound) and downstream (inbound) traffic
		upstream := int64(flow.CountersOrig.Bytes)
		downstream := int64(flow.CountersReply.Bytes)

		// Update stats for source IP
		if srcIP.IsValid() && !srcIP.IsUnspecified() && !srcIP.IsMulticast() {
			if !lstat.isLanIP(srcIP) {
				continue
			}
			lstat.addSpeeds(srcIP.String(), upstream, downstream)
		}

		// Update stats for destination IP
		if dstIP.IsValid() && !dstIP.IsUnspecified() && !dstIP.IsMulticast() {
			if !lstat.isLanIP(dstIP) {
				continue
			}
			lstat.addSpeeds(dstIP.String(), downstream, upstream)
		}
	}

	curr := time.Now()
	for _, host := range lstat.hosts {
		rxTotal, txTotal := host.rxTotal, host.txTotal
		host.rxTotal, host.txTotal = 0, 0
		_ = lstat.processHost(host, curr, txTotal, rxTotal)
		//if host.ip == testIP {
		//	l.Debugln("item1=", len(host.items), "item2=", len(host2.items))
		//}
	}
	return nil
}

func (lstat *LanStats) addSpeeds(ip string, tx, rx int64) {
	host, ok := lstat.hosts[ip]
	if !ok {
		host = &LanStatHost{
			ip:      ip,
			rxTotal: rx,
			txTotal: tx,
		}
		lstat.hosts[ip] = host
	} else {
		host.rxTotal += rx
		host.txTotal += tx
	}
}

func (lstat *LanStats) processHost(host *LanStatHost, curr time.Time, tx, rx int64) *LanStatHost {
	//if host.ip == testIP {
	//	l.Debugln("conntrack ip=", host.ip, "tx=", tx, "rx=", rx)
	//}
	if host.lastItem == nil {
		host.lastItem = &NetworkStatisticsItem{
			txStart:   tx,
			rxStart:   rx,
			startTime: curr,
		}
		host.items = make([]*NetworkStatisticsItem, 0, slots+1)
		return host
	}
	item := host.lastItem
	item.endTime = curr
	duration := item.endTime.Sub(item.startTime).Milliseconds() + 1
	if tx < item.txStart {
		if len(host.items) > 0 {
			item.txAvg = host.items[len(host.items)-1].txAvg
		}
	} else {
		item.txAvg = 1000 * (tx - item.txStart) / int64(duration)
	}

	if rx < item.rxStart {
		if len(host.items) > 0 {
			item.rxAvg = host.items[len(host.items)-1].rxAvg
		}
	} else {
		item.rxAvg = 1000 * (rx - item.rxStart) / int64(duration)
	}

	if len(host.items) >= slots {
		for i := 0; i < slots-1; i++ {
			host.items[i] = host.items[i+1]
		}
		host.items = host.items[:slots-1]
	}
	host.items = append(host.items, item)

	host.lastItem = &NetworkStatisticsItem{
		startTime: curr,
		rxStart:   rx,
		txStart:   tx,
	}
	return host
}

func (lstat *LanStats) respHosts(req *fetchHostStatsReq) error {
	if req.hostIP != "" {
		rets := make([]*LanHostRet, 0, 1)
		host, ok := lstat.hosts[req.hostIP]
		if ok {
			//l.Debugln("found host ip=", host.ip, "items=", len(host.items))
			ret := &LanHostRet{
				ip: host.ip,
			}
			if req.speedOnly {
				ret.items = make([]*NetworkStatisticsItem, 0, 1)
				if len(host.items) > 0 {
					ret.items = append(ret.items, host.items[len(host.items)-1])
				}
			} else {
				ret.items = make([]*NetworkStatisticsItem, len(host.items))
				copy(ret.items, host.items)
			}
			rets = append(rets, ret)
		}
		req.itemCh <- rets
		close(req.itemCh)
		return nil
	}

	// Get all items speeds
	rets := make([]*LanHostRet, 0, len(lstat.hosts))
	for _, host := range lstat.hosts {
		ret := &LanHostRet{
			ip: host.ip,
		}
		if req.speedOnly {
			// Get last speed only
			ret.items = make([]*NetworkStatisticsItem, 0, 1)
			if len(host.items) > 0 {
				ret.items = append(ret.items, host.items[len(host.items)-1])
			}
		} else {
			ret.items = make([]*NetworkStatisticsItem, len(host.items))
			copy(ret.items, host.items)
		}
		rets = append(rets, ret)
	}
	req.itemCh <- rets
	close(req.itemCh)
	return nil
}

func (lstat *LanStats) isLanIP(ip netip.Addr) bool {
	// TODO check if it's LAN IP?
	return true
}

func (lstat *LanStats) reqHosts(hostIP string, speedOnly bool) []*LanHostRet {
	req := &fetchHostStatsReq{
		hostIP:    hostIP,
		speedOnly: speedOnly,
		itemCh:    make(chan []*LanHostRet, 1),
	}
	lstat.fetchHostStatsCh <- req
	rets := <-req.itemCh
	return rets
}
