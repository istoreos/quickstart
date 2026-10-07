//go:build linux

package service

import (
	"bufio"
	"context"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/istoreos/quickstart/backend/models"
	"github.com/ti-mo/conntrack"
)

const (
	testIP                  = "192.168.9.3"
	lanStatTimeTick         = 3 * time.Second
	lanPrefixRefreshEvery   = time.Minute
	defaultLanStatHostTTL   = 10 * time.Minute
	defaultLanStatHostLimit = 2048
)

type LanStats struct {
	tm        *time.Timer
	c         *conntrack.Conn
	isRunning bool
	// Auto stop when no request after 30s
	lastRunningTime       time.Time
	hosts                 map[string]*LanStatHost
	fetchHostStatsCh      chan *fetchHostStatsReq
	lanPrefixes           []netip.Prefix
	lanPrefixReader       func() ([]netip.Prefix, error)
	lastPrefixRead        time.Time
	maxHosts              int
	hostIdleTTL           time.Duration
	now                   func() time.Time
	lastFetchAt           time.Time
	lastFetchErr          error
	sampleCount           int64
	procFallback          bool
	lastNetlinkAttempt    time.Time
	diagnosticSampling    atomic.Bool
	diagnosticFallback    atomic.Bool
	diagnosticHostCount   atomic.Int64
	diagnosticFlowCount   atomic.Int64
	diagnosticSampleNanos atomic.Int64
	diagnosticFailures    atomic.Int64
	diagnosticEvictions   atomic.Int64
	diagnosticSamples     atomic.Int64
}

type LanStatHost struct {
	ip              string
	rxTotal         int64
	txTotal         int64
	lastItem        *NetworkStatisticsItem
	items           []*NetworkStatisticsItem
	lastSeen        time.Time
	txBytes         int64
	rxBytes         int64
	connectionCount int64
	sampledAt       time.Time
}

type fetchHostStatsReq struct {
	hostIP    string
	speedOnly bool
	response  chan lanStatsSnapshot
}

func NewLanStats() *LanStats {
	stats := &LanStats{
		hosts:            make(map[string]*LanStatHost),
		fetchHostStatsCh: make(chan *fetchHostStatsReq, 1),
		lanPrefixReader:  readSystemLANPrefixes,
		maxHosts:         defaultLanStatHostLimit,
		hostIdleTTL:      defaultLanStatHostTTL,
		now:              time.Now,
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
		lstat.diagnosticSampling.Store(true)
		err := lstat.fetchItems()
		lstat.lastFetchErr = err
		if err == nil {
			lstat.lastFetchAt = lstat.currentTime()
			lstat.sampleCount++
			lstat.diagnosticSamples.Add(1)
		}
		if err != nil {
			lstat.diagnosticFailures.Add(1)
			l.Debugln("fetchItems in fetchStatCh err=", err)
		}

		// Return the news
		lstat.respHosts(req)

	case <-tm.C:
		now := time.Now()
		if lstat.lastRunningTime.Add(10 * lanStatTimeTick).Before(now) {
			lstat.isRunning = false
			lstat.diagnosticSampling.Store(false)
			tm.Stop()
			lstat.closeConn()
			return nil
		}
		err := lstat.fetchItems()
		lstat.lastFetchErr = err
		if err == nil {
			lstat.lastFetchAt = lstat.currentTime()
			lstat.sampleCount++
			lstat.diagnosticSamples.Add(1)
		}
		if err != nil {
			lstat.diagnosticFailures.Add(1)
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
	started := time.Now()
	flowCount := int64(0)
	defer func() {
		lstat.diagnosticFlowCount.Store(flowCount)
		lstat.diagnosticHostCount.Store(int64(len(lstat.hosts)))
		lstat.diagnosticSampleNanos.Store(time.Since(started).Nanoseconds())
	}()
	lstat.refreshLANPrefixes()
	for _, host := range lstat.hosts {
		host.connectionCount = 0
	}
	err := lstat.walkTrafficFlows(func(flow lanTrafficFlow) {
		flowCount++
		srcIP := flow.source
		dstIP := flow.destination
		// Skip broadcast IPs (like 255.255.255.255)
		if srcIP.IsUnspecified() || dstIP.IsUnspecified() ||
			srcIP.IsMulticast() || dstIP.IsMulticast() {
			return
		}
		lstat.recordFlow(srcIP, dstIP, flow.upstream, flow.downstream)
	})
	if err != nil {
		return err
	}

	curr := lstat.currentTime()
	lstat.pruneHosts(curr)
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

type lanTrafficFlow struct {
	source      netip.Addr
	destination netip.Addr
	upstream    int64
	downstream  int64
}

func (lstat *LanStats) walkTrafficFlows(visit func(lanTrafficFlow)) error {
	now := lstat.currentTime()
	tryNetlink := !lstat.procFallback || lstat.lastNetlinkAttempt.IsZero() || now.Sub(lstat.lastNetlinkAttempt) >= time.Minute
	if tryNetlink {
		lstat.lastNetlinkAttempt = now
		if err := lstat.dial(); err == nil {
			flows, dumpErr := lstat.c.Dump(nil)
			if dumpErr == nil {
				for _, flow := range flows {
					visit(lanTrafficFlow{
						source: flow.TupleOrig.IP.SourceAddress, destination: flow.TupleOrig.IP.DestinationAddress,
						upstream: int64(flow.CountersOrig.Bytes), downstream: int64(flow.CountersReply.Bytes),
					})
				}
				lstat.procFallback = false
				lstat.diagnosticFallback.Store(false)
				return nil
			}
		}
		lstat.closeConn()
		lstat.procFallback = true
		lstat.diagnosticFallback.Store(true)
	}
	return walkProcConntrackFlows("/proc/net/nf_conntrack", visit)
}

func walkProcConntrackFlows(path string, visit func(lanTrafficFlow)) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 1024*1024)
	for scanner.Scan() {
		var source, destination netip.Addr
		var upstream, downstream int64
		byteCount := 0
		for _, field := range strings.Fields(scanner.Text()) {
			key, value, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			switch key {
			case "src":
				if !source.IsValid() {
					source, _ = netip.ParseAddr(value)
				}
			case "dst":
				if !destination.IsValid() {
					destination, _ = netip.ParseAddr(value)
				}
			case "bytes":
				if byteCount < 2 {
					parsed, parseErr := strconv.ParseInt(value, 10, 64)
					if parseErr == nil && parsed >= 0 {
						if byteCount == 0 {
							upstream = parsed
						} else {
							downstream = parsed
						}
						byteCount++
					}
				}
			}
		}
		if !source.IsValid() || !destination.IsValid() || byteCount == 0 {
			continue
		}
		visit(lanTrafficFlow{source: source.Unmap(), destination: destination.Unmap(), upstream: upstream, downstream: downstream})
	}
	return scanner.Err()
}

func (lstat *LanStats) recordFlow(srcIP, dstIP netip.Addr, upstream, downstream int64) {
	if lstat.isLanIP(srcIP) {
		lstat.addSpeeds(srcIP.String(), upstream, downstream)
	}
	if lstat.isLanIP(dstIP) {
		lstat.addSpeeds(dstIP.String(), downstream, upstream)
	}
}

func (lstat *LanStats) addSpeeds(ip string, tx, rx int64) {
	now := lstat.currentTime()
	host, ok := lstat.hosts[ip]
	if !ok {
		lstat.pruneHosts(now)
		if len(lstat.hosts) >= lstat.hostLimit() {
			lstat.evictOldestHost()
		}
		host = &LanStatHost{
			ip:       ip,
			rxTotal:  rx,
			txTotal:  tx,
			lastSeen: now,
		}
		lstat.hosts[ip] = host
	} else {
		host.rxTotal += rx
		host.txTotal += tx
		host.lastSeen = now
	}
	host.connectionCount++
}

func (lstat *LanStats) currentTime() time.Time {
	if lstat.now != nil {
		return lstat.now()
	}
	return time.Now()
}

func (lstat *LanStats) hostLimit() int {
	if lstat.maxHosts > 0 {
		return lstat.maxHosts
	}
	return defaultLanStatHostLimit
}

func (lstat *LanStats) idleTTL() time.Duration {
	if lstat.hostIdleTTL > 0 {
		return lstat.hostIdleTTL
	}
	return defaultLanStatHostTTL
}

func (lstat *LanStats) pruneHosts(now time.Time) {
	for ip, host := range lstat.hosts {
		if !host.lastSeen.IsZero() && now.Sub(host.lastSeen) > lstat.idleTTL() {
			delete(lstat.hosts, ip)
		}
	}
	for len(lstat.hosts) > lstat.hostLimit() {
		lstat.evictOldestHost()
	}
}

func (lstat *LanStats) evictOldestHost() {
	var oldestIP string
	var oldestTime time.Time
	for ip, host := range lstat.hosts {
		if oldestIP == "" || host.lastSeen.Before(oldestTime) || (host.lastSeen.Equal(oldestTime) && ip < oldestIP) {
			oldestIP = ip
			oldestTime = host.lastSeen
		}
	}
	if oldestIP != "" {
		delete(lstat.hosts, oldestIP)
		lstat.diagnosticEvictions.Add(1)
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
		host.sampledAt = curr
		return host
	}
	item := host.lastItem
	item.endTime = curr
	duration := item.endTime.Sub(item.startTime).Milliseconds() + 1
	if tx < item.txStart {
		item.txAvg = 0
	} else {
		delta := tx - item.txStart
		item.txAvg = 1000 * delta / int64(duration)
		host.txBytes += delta
	}

	if rx < item.rxStart {
		item.rxAvg = 0
	} else {
		delta := rx - item.rxStart
		item.rxAvg = 1000 * delta / int64(duration)
		host.rxBytes += delta
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
	host.sampledAt = curr
	return host
}

func (lstat *LanStats) respHosts(req *fetchHostStatsReq) error {
	if req.hostIP != "" {
		rets := make([]*LanHostRet, 0, 1)
		host, ok := lstat.hosts[req.hostIP]
		if ok {
			//l.Debugln("found host ip=", host.ip, "items=", len(host.items))
			ret := &LanHostRet{
				ip: host.ip, txBytes: host.txBytes, rxBytes: host.rxBytes,
				connectionCount: host.connectionCount, sampledAt: host.sampledAt,
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
		req.response <- lanStatsSnapshot{hosts: rets, sampledAt: lstat.lastFetchAt, err: lstat.lastFetchErr, samples: lstat.sampleCount}
		close(req.response)
		return nil
	}

	// Get all items speeds
	rets := make([]*LanHostRet, 0, len(lstat.hosts))
	for _, host := range lstat.hosts {
		ret := &LanHostRet{
			ip: host.ip, txBytes: host.txBytes, rxBytes: host.rxBytes,
			connectionCount: host.connectionCount, sampledAt: host.sampledAt,
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
	req.response <- lanStatsSnapshot{hosts: rets, sampledAt: lstat.lastFetchAt, err: lstat.lastFetchErr, samples: lstat.sampleCount}
	close(req.response)
	return nil
}

func (lstat *LanStats) isLanIP(ip netip.Addr) bool {
	if !ip.IsValid() || ip.IsUnspecified() || ip.IsMulticast() {
		return false
	}
	ip = ip.Unmap()
	for _, prefix := range lstat.lanPrefixes {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func (lstat *LanStats) refreshLANPrefixes() {
	if lstat.lanPrefixReader == nil {
		return
	}
	now := lstat.currentTime()
	if !lstat.lastPrefixRead.IsZero() && now.Sub(lstat.lastPrefixRead) < lanPrefixRefreshEvery {
		return
	}
	lstat.lastPrefixRead = now
	prefixes, err := lstat.lanPrefixReader()
	if err != nil {
		l.Debugln("read LAN prefixes failed, err=", err)
		return
	}
	if len(prefixes) == 0 {
		l.Debugln("read LAN prefixes returned no usable network")
		return
	}
	lstat.lanPrefixes = prefixes
}

func readSystemLANPrefixes() ([]netip.Prefix, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	status, err := ubusGetLanStatus(ctx)
	if err != nil {
		return nil, err
	}
	return lanPrefixesFromStatus(status), nil
}

func lanPrefixesFromStatus(status *ubusLanStatus) []netip.Prefix {
	if status == nil {
		return nil
	}
	prefixes := make([]netip.Prefix, 0, len(status.network.Ipv4)+len(status.network.Ipv6)+len(status.network.Ipv6PA))
	seen := make(map[netip.Prefix]struct{})
	appendAddress := func(address *ubusNetworkInterfaceAddress) {
		if address == nil {
			return
		}
		addr, err := netip.ParseAddr(address.Address)
		if err != nil || address.Mask < 0 || address.Mask > addr.BitLen() {
			return
		}
		prefix := netip.PrefixFrom(addr.Unmap(), address.Mask).Masked()
		if _, ok := seen[prefix]; ok {
			return
		}
		seen[prefix] = struct{}{}
		prefixes = append(prefixes, prefix)
	}
	for _, address := range status.network.Ipv4 {
		appendAddress(address)
	}
	for _, address := range status.network.Ipv6 {
		appendAddress(address)
	}
	for _, assignment := range status.network.Ipv6PA {
		if assignment != nil {
			appendAddress(&ubusNetworkInterfaceAddress{Address: assignment.Address, Mask: assignment.Mask})
		}
	}
	return prefixes
}

func (lstat *LanStats) reqHosts(hostIP string, speedOnly bool) []*LanHostRet {
	return lstat.reqSnapshotContext(context.Background(), hostIP, speedOnly).hosts
}

func (lstat *LanStats) diagnostics() *models.DeviceSamplerRuntimeDiagnostics {
	if lstat == nil {
		return &models.DeviceSamplerRuntimeDiagnostics{}
	}
	return &models.DeviceSamplerRuntimeDiagnostics{
		Sampling:      lstat.diagnosticSampling.Load(),
		ProcFallback:  lstat.diagnosticFallback.Load(),
		HostCount:     lstat.diagnosticHostCount.Load(),
		FlowCount:     lstat.diagnosticFlowCount.Load(),
		LastSampleMS:  time.Duration(lstat.diagnosticSampleNanos.Load()).Milliseconds(),
		SampleCount:   lstat.diagnosticSamples.Load(),
		FailureCount:  lstat.diagnosticFailures.Load(),
		EvictionCount: lstat.diagnosticEvictions.Load(),
	}
}

func (lstat *LanStats) reqSnapshot(hostIP string, speedOnly bool) lanStatsSnapshot {
	return lstat.reqSnapshotContext(context.Background(), hostIP, speedOnly)
}

func (lstat *LanStats) reqSnapshotContext(ctx context.Context, hostIP string, speedOnly bool) lanStatsSnapshot {
	req := &fetchHostStatsReq{
		hostIP:    hostIP,
		speedOnly: speedOnly,
		response:  make(chan lanStatsSnapshot, 1),
	}
	select {
	case <-ctx.Done():
		return lanStatsSnapshot{err: ctx.Err()}
	case lstat.fetchHostStatsCh <- req:
	}
	select {
	case <-ctx.Done():
		return lanStatsSnapshot{err: ctx.Err()}
	case response := <-req.response:
		return response
	}
}
