package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/digineo/go-uci"
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/dhns"
	"github.com/linkease/quick-start/istore-backend/models"
	dhnsconflict "github.com/linkease/quick-start/istore-backend/modules/dhns/conflict"
	dhnsevents "github.com/linkease/quick-start/istore-backend/modules/dhns/events"
	dhnshttpresult "github.com/linkease/quick-start/istore-backend/modules/dhns/httpresult"
	dhnsnetns "github.com/linkease/quick-start/istore-backend/modules/dhns/netns"
	dhnsnetsections "github.com/linkease/quick-start/istore-backend/modules/dhns/netsections"
	dhnsrecovery "github.com/linkease/quick-start/istore-backend/modules/dhns/recovery"
	dhnsudhcp "github.com/linkease/quick-start/istore-backend/modules/dhns/udhcp"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type hijackerConn struct {
	net.Conn
	tr io.Reader
}

func (c hijackerConn) Read(b []byte) (n int, err error) {
	n, err = c.tr.Read(b)
	return
}

func (backend *ServiceBackend) setupDhns() {
	uci.LoadConfig("quickstart", false)
	if val, ok := uci.GetLast("quickstart", "main", "disable_dhns"); ok && val == "1" {
		l.Debugln("disable DHNS")
		backend.disableDHNS = true
		backend.stopUdhcpc()
		backend.stopDhns()
	} else {
		backend.dhnsServer = dhns.NewDhnsServer()
	}
}

func (backend *ServiceBackend) DhnsDisabled() bool {
	return backend.disableDHNS
}

func (backend *ServiceBackend) DhnsConnect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	c, bufrw, err := hj.Hijack()
	if err != nil {
		return
	}
	bufrw.Flush()

	left := bufrw.Reader.Buffered()
	if left > 0 {
		c = hijackerConn{
			Conn: c,
			tr:   bufrw,
		}
	}
	l.Debugln("dhns connect in")
	backend.dhnsServer.HandleConn(c)
}

func (backend *ServiceBackend) DhnsProxy(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id := r.Header.Get("Proxyid")
	if id == "" {
		w.Header().Add("Connection", "close")
		http.Error(w, "proxy error", http.StatusBadRequest)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	c, bufrw, err := hj.Hijack()
	if err != nil {
		return
	}
	bufrw.Flush()

	left := bufrw.Reader.Buffered()
	if left > 0 {
		c = hijackerConn{
			Conn: c,
			tr:   bufrw,
		}
	}
	l.Debugln("dhns proxy in")
	backend.dhnsServer.PutDhnsConn(id, c)
}

func (backend *ServiceBackend) DhnsForward(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	addr := r.Header.Get("TargetAddr")
	if addr == "" {
		w.Header().Add("Connection", "close")
		http.Error(w, "target addr error", http.StatusBadRequest)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	c, bufrw, err := hj.Hijack()
	if err != nil {
		return
	}
	bufrw.Flush()

	left := bufrw.Reader.Buffered()
	if left > 0 {
		c = hijackerConn{
			Conn: c,
			tr:   bufrw,
		}
	}
	forward(c, addr)
}

func forward(local net.Conn, remoteAddr string) {
	defer local.Close()
	//l.Debugln("forward target=", remoteAddr)
	remote, err := net.DialTimeout("tcp", remoteAddr, time.Second*10)
	if err != nil {
		l.Debugf("remote dial failed: %v\n", err)
		return
	}
	defer remote.Close()
	p1die := make(chan struct{})
	p2die := make(chan struct{})
	go func() {
		io.Copy(local, remote)
		close(p1die)
	}()
	go func() {
		io.Copy(remote, local)
		close(p2die)
	}()
	select {
	case <-p1die:
	case <-p2die:
	}
}

type DhnsChangeInfo = models.DHNSChangeRequest

func (backend *ServiceBackend) HandleDhnsChange(evt DhnsChangeInfo) bool {
	if !dhnsevents.ShouldTriggerIfaceEvent(evt) {
		return false
	}
	backend.sentIfaceEvent()
	return true
}

func (backend *ServiceBackend) sentIfaceEvent() {
	if backend.dhnsState.MarkIfaceChange() {
		go backend.dhnsChanging()
	}
}

func (backend *ServiceBackend) dhnsChanging() {
	defer backend.dhnsState.Finish()
	time.Sleep(time.Second * 3)

	for {
		var netSecs map[string]*networkSecInfo
		var wanSec, lanSec, planbSec *networkSecInfo
		pending := backend.dhnsState.Drain()
		ifaceChange := pending.IfaceChange
		dhcpChange := pending.DHCPChange
		dhcpGot := pending.DHCP
		if !ifaceChange && !dhcpChange {
			return
		}
		l.Debugln("ifaceChange=", ifaceChange, "dhcpChange=", dhcpChange)

		// Ignore dhcp change when the network is OK
		if !ifaceChange {
			if dhnsIsNetworkOK() {
				continue
			}
		}

		uci.LoadConfig("network", true)
		netSecs = dhnsGetNetSections()
		for sec, secInfo := range netSecs {
			if sec == "lan" {
				lanSec = secInfo
				lanSec.Up = backend.getIfaceStatus("lan")
			} else if sec == "planb" {
				planbSec = secInfo
				planbSec.Up = backend.getIfaceStatus("planb")
			} else if sec == "wan" {
				wanSec = secInfo
				wanSec.Up = backend.getIfaceStatus("wan")
			}
			//l.Debugln("sec=", sec, "info=", secInfo.Device, secInfo.IPNet.IP.String(), net.IP(secInfo.IPNet.Mask).String(), secInfo.Gateway, secInfo.Proto)
		}
		if lanSec == nil {
			l.Debugln("lan is not found")
			continue
		}

		if ifaceChange {
			networkOK := false
			if !(wanSec != nil && wanSec.Up) && !(lanSec.Proto == "dhcp" || (lanSec.Proto == "static" && lanSec.Gateway != "")) {
				networkOK = dhnsIsNetworkOK()
			}
			ifaceDecision := dhnsrecovery.PlanIfaceChange(dhnsrecovery.IfaceChangeInput{
				LAN:       toDhnsRecoverySection(lanSec),
				WAN:       toDhnsRecoverySectionPtr(wanSec),
				PlanB:     toDhnsRecoverySectionPtr(planbSec),
				NetworkOK: networkOK,
			})
			if ifaceDecision.StopUdhcpc {
				backend.stopUdhcpc()
			}
			if ifaceDecision.IfdownPlanB {
				utils.BatchRun(context.TODO(), []string{"ifdown planb"}, 0)
			}
			if ifaceDecision.IfdownPlanBAndIfupLAN {
				utils.BatchRun(context.TODO(), []string{"ifdown planb && ifup lan"}, 0)
			} else if ifaceDecision.IfupLAN {
				utils.BatchRun(context.TODO(), []string{"ifup lan"}, 0)
			}
			if ifaceDecision.CheckLANHasIP {
				go func() {
					time.Sleep(time.Second * 3)
					for i := 0; i < 2; i++ {
						_, err := utils.GetInterfaceIpv4(lanSec.Device)
						if err == nil {
							return
						}
						l.Debugln("lan have not IP")
						time.Sleep(time.Second * 2)
					}
					// Try up again if still have not IP in lan
					utils.BatchRun(context.TODO(), []string{"ifup lan"}, 0)
				}()
			}
			if ifaceDecision.CheckWANLANConflict {
				if dhnsConflictWanAndLan(wanSec, lanSec) {
					backend.setupStaticIPInOtherNS(lanSec.IPNet)
				} else {
					backend.stopDhns()
				}
			} else if ifaceDecision.SetupStaticIPInOtherNS {
				backend.setupStaticIPInOtherNS(lanSec.IPNet)
			} else if ifaceDecision.StopDhns {
				backend.stopDhns()
			}
			if ifaceDecision.StartUdhcpc {
				backend.startUdhcpc(lanSec.Device)
			} else {
				continue
			}
		}
		if dhcpChange {
			l.Debugln("dhcpChange dhcpInfo=", dhcpGot)

			if !backend.startDhns() {
				continue
			}

			var planBRecoverySec *dhnsrecovery.Section
			if planbSec != nil {
				planBRecoverySec = &dhnsrecovery.Section{
					Device: planbSec.Device,
					IPNet:  planbSec.IPNet,
					Up:     planbSec.Up,
				}
			}
			recoveryDecision := dhnsrecovery.PlanDHCPChange(dhnsrecovery.DHCPChangeInput{
				LAN: dhnsrecovery.Section{
					Device: lanSec.Device,
					IPNet:  lanSec.IPNet,
					Up:     lanSec.Up,
				},
				PlanB: planBRecoverySec,
				DHCP: dhnsrecovery.DHCPLease{
					IP:      dhcpGot.Ip,
					Subnet:  dhcpGot.Subnet,
					Gateway: dhcpGot.Gateway,
				},
			})
			if !recoveryDecision.Valid {
				continue
			}
			if recoveryDecision.Conflict {
				// Conflict
				if recoveryDecision.RestartLAN {
					utils.BatchRun(context.TODO(), []string{"ifdown planb && ifup lan"}, 10)
				}
				continue
			}

			// The dhns instance is alive, check network
			if !backend.dhnsSetupNetwork(&dhns.DhnsStatic{
				IP:      dhcpGot.Ip,
				Mask:    dhcpGot.Subnet,
				Gateway: dhcpGot.Gateway,
			}) {
				// Check network failed
				l.Debugln("check network failed")
				continue
			}
			// Check network again
			if dhnsIsNetworkOK() {
				backend.stopUdhcpc()
				continue
			}

			/* 删除此行，将planb加入防火墙LAN区域
			// 将planb加入防火墙LAN区域
			utils.BatchRun(context.TODO(), []string{
				"LAN_ZONE=`uci show firewall | grep -E '^firewall\\.@zone\\[[0-9]+\\]\\.name='\"'lan'\" | head -n1 | head -c -12`",
				"if [ -n \"$LAN_ZONE\" ]; then",
				"	uci get \"$LAN_ZONE.network\" | grep -Fwq planb || {",
				"		uci add_list \"$LAN_ZONE.network=planb\"",
				"		uci commit firewall",
				"	}",
				"fi",
			}, 0)
			/**/

			cmdList := append([]string(nil), recoveryDecision.PlanBUCICommands...)

			//l.Debugln("cmdList=", cmdList)
			// Stop udhcpc first
			backend.stopUdhcpc()
			if len(cmdList) > 0 {
				cmdList = append(cmdList, "commit network")
				utils.UCIBatchRun(context.TODO(), cmdList, "/etc/init.d/network reload", 0)
			}
			cmdList = cmdList[:0]
			if recoveryDecision.IfupPlanB {
				cmdList = append(cmdList, "ifup planb")
			}
			utils.BatchRun(context.TODO(), []string{strings.Join(cmdList, " && ")}, 10)
			// Ip conflict or the lan is OK, dhns is not need
			backend.stopDhns()
		}
	}
}

type networkSecInfo = dhnsnetsections.Section

/*
subnet=255.255.255.0
router=192.168.16.1
interface=br-lan
dns=192.168.16.1
siaddr=192.168.16.1
serverid=192.168.16.1
broadcast=192.168.16.255
ip=192.168.16.150
lease=86400
mask=24
*/
type DhcpInfo = models.DHNSDhcpValidRequest

func dhnsIsNetworkOK() bool {
	c, err := net.DialTimeout("tcp", "114.114.114.114:53", time.Second*dialTimeout)
	closeConn(c)
	if err != nil {
		// check again
		c, err = net.DialTimeout("tcp", "223.5.5.5:53", time.Second*dialTimeout)
		closeConn(c)
	}
	if err != nil {
		return false
	}
	return true
}

func dhnsGetNetSections() map[string]*networkSecInfo {
	return dhnsnetsections.Collect(dhnsUCIReader{}, utils.GetInterfaceIpv4)
}

type dhnsUCIReader struct{}

func (dhnsUCIReader) Sections(config string, sectionType string) ([]string, bool) {
	return uci.GetSections(config, sectionType)
}

func (dhnsUCIReader) Last(config string, section string, option string) (string, bool) {
	return uci.GetLast(config, section, option)
}

type ifaceStatus struct {
	Up bool `json:"up"`
}

func (backend *ServiceBackend) getIfaceStatus(iface string) bool {
	var status ifaceStatus
	err := UbusCallWithObject(context.TODO(), fmt.Sprintf("network.interface status {\"interface\":\"%s\"}", iface), &status)
	if err == nil {
		return status.Up
	}
	return false
}

func (backend *ServiceBackend) startUdhcpc(lanDev string) {
	data, err := ioutil.ReadFile(dhnsudhcp.PIDFile)
	pid := dhnsudhcp.PIDFromFileData(data, err)
	var find bool
	if pid > 0 {
		find, _ = utils.PidExists(pid)
	}
	plan := dhnsudhcp.PlanStart(lanDev, pid, find)
	if plan.AlreadyRunning {
		return
	}
	if plan.RemovePIDFile {
		os.Remove(dhnsudhcp.PIDFile)
	}
	var wait bool
	dhcpGotCh := make(chan struct{}, 1)
	if backend.dhnsState.RegisterDHCPWaiter(dhcpGotCh) {
		wait = true
	}
	if wait {
		go func() {
			tick := time.NewTimer(time.Second * 15)
			defer tick.Stop()
			select {
			case <-dhcpGotCh:
				// Got dhcp event
				l.Debugln("got dhcp event")
			case <-tick.C:
				// Timeout, sent ifaceChange event again
				l.Debugln("dhcp event timeout")
				backend.dhnsState.ClearDHCPWaiter()
				backend.sentIfaceEvent()
			}
		}()
	}
	utils.BatchRun(context.TODO(), plan.Commands, 10)
}

func (backend *ServiceBackend) stopUdhcpc() {
	data, err := ioutil.ReadFile(dhnsudhcp.PIDFile)
	pid := dhnsudhcp.PIDFromFileData(data, err)
	var find bool
	if pid > 0 {
		find, _ = utils.PidExists(pid)
	}
	plan := dhnsudhcp.PlanStop(pid, find)
	if len(plan.Commands) > 0 {
		utils.BatchRun(context.TODO(), plan.Commands, 5)
	}
	if plan.RemovePIDFile {
		os.Remove(dhnsudhcp.PIDFile)
	}
	backend.dhnsState.ResetDHCP()
}

func dhnsNoConflictIP(netIP1, netIP2 net.IPNet) string {
	return dhnsconflict.NoConflictIP(netIP1, netIP2)
}

func dhnsConflictWanAndLan(wanSec, lanSec *networkSecInfo) (needDhns bool) {
	plan := dhnsconflict.PlanWANLANConflict(
		toDhnsConflictSection(wanSec),
		toDhnsConflictSection(lanSec),
		dhnsLANDHCPDisabled(),
	)
	if len(plan.NetworkUCICommands) > 0 {
		reloadCommand := ""
		if plan.ReloadNetwork {
			reloadCommand = "/etc/init.d/network reload"
		}
		utils.UCIBatchRun(context.TODO(), plan.NetworkUCICommands, reloadCommand, 0)
	}
	return plan.NeedDHNS
}

func dhnsLANDHCPDisabled() bool {
	out, err := utils.BatchOutput(context.TODO(), []string{"uci -q get dhcp.lan.ignore"}, 0)
	return err == nil && strings.Contains(string(out), "1")
}

func toDhnsConflictSection(sec *networkSecInfo) dhnsconflict.NetworkSection {
	if sec == nil {
		return dhnsconflict.NetworkSection{}
	}
	return dhnsconflict.NetworkSection{
		Section: sec.Name,
		Proto:   sec.Proto,
		Gateway: sec.Gateway,
		IPNet:   sec.IPNet,
	}
}

func toDhnsRecoverySectionPtr(sec *networkSecInfo) *dhnsrecovery.Section {
	if sec == nil {
		return nil
	}
	section := toDhnsRecoverySection(sec)
	return &section
}

func toDhnsRecoverySection(sec *networkSecInfo) dhnsrecovery.Section {
	if sec == nil {
		return dhnsrecovery.Section{}
	}
	return dhnsrecovery.Section{
		Device:  sec.Device,
		Proto:   sec.Proto,
		Gateway: sec.Gateway,
		IPNet:   sec.IPNet,
		Up:      sec.Up,
	}
}

func (backend *ServiceBackend) setupStaticIPInOtherNS(lanIP net.IPNet) {
	ip1 := net.ParseIP("192.168.100.1")
	ip2 := net.ParseIP("192.168.101.1")
	var ip net.IP
	if !lanIP.Contains(ip1) {
		ip = ip1
	} else if !lanIP.Contains(ip2) {
		ip = ip2
	} else {
		// The IP is conflict with dhns, stop it
		backend.stopDhns()
		return
	}
	if backend.startDhns() {
		backend.dhnsSetupNetwork(&dhns.DhnsStatic{
			IP:   ip.String(),
			Mask: net.IP(lanIP.Mask).String(),
		})
	}
}

func (backend *ServiceBackend) HandleDhcpValid(info DhcpInfo) {
	l.Debugln("got dhcp info=", info)
	start, dhcpGotCh := backend.dhnsState.MarkDHCPValid(info)
	if dhcpGotCh != nil {
		select {
		case dhcpGotCh <- struct{}{}:
			// Notify changed
		default:
		}
	}
	if start {
		go backend.dhnsChanging()
	}
}

func (backend *ServiceBackend) startDhns() bool {
	if !backend.dhnsAlive() {
		if err := createDhnsInterface(); err != nil {
			l.Debugln("create dhns interface err=", err)
		}
		utils.BatchRun(context.TODO(), dhnsnetns.StartCommands(), 0)
		var alive bool
		for i := 0; i < 3; i++ {
			if backend.dhnsAlive() {
				alive = true
				break
			}
			time.Sleep(time.Second)
		}
		if !alive {
			return false
		}
	}
	return true
}

func createDhnsInterface() error {
	_, err := net.InterfaceByName(dhnsnetns.VethHostDevice)
	if err != nil {
		output, err := utils.BatchOutput(context.TODO(), dhnsnetns.CreateInterfaceCommands(), 5)
		if err := dhnsnetns.ValidateCreateInterfaceOutput(output, err); err != nil {
			return err
		}
	}
	output, err := utils.BatchOutput(context.TODO(), dhnsnetns.BridgePortsQueryCommands(), 5)
	bridgeCommands, err := dhnsnetns.PlanBridgePortCommands(output, err)
	if err != nil {
		return err
	}
	if len(bridgeCommands) > 0 {
		utils.BatchRun(context.TODO(), bridgeCommands, 10)
	}
	return nil
}

func (backend *ServiceBackend) dhnsAlive() bool {
	resp, err := backend.dhnsServer.Client().Get("http://localhost/api/dhns/alive/")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	return dhnshttpresult.IsOK(resp.StatusCode, data, err)
}

func (backend *ServiceBackend) stopDhns() {
	//l.Debugln("stopDhns")
	utils.BatchRun(context.TODO(), dhnsnetns.StopCommands(), 5)
}

func (backend *ServiceBackend) dhnsSetupNetwork(staticNet *dhns.DhnsStatic) bool {
	data, err := json.Marshal(staticNet)
	if err != nil {
		return false
	}
	resp, err := backend.dhnsServer.Client().Post("http://localhost/api/dhns/static/", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	return dhnshttpresult.IsOK(resp.StatusCode, data, err)
}

func (backend *ServiceBackend) DhnsTest(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("OK"))
	if strings.HasSuffix(r.URL.Path, "/startdhns/") {
		backend.startDhns()
	}
	if strings.HasSuffix(r.URL.Path, "/stopdhns/") {
		backend.stopDhns()
	}
	if strings.HasSuffix(r.URL.Path, "/startdhcp/") {
		backend.startUdhcpc("br-lan")
	}
	if strings.HasSuffix(r.URL.Path, "/stopdhcp/") {
		backend.stopUdhcpc()
	}
	if strings.HasSuffix(r.URL.Path, "/iface/") {
		uci.LoadConfig("network", true)
		netSecs := dhnsGetNetSections()
		for sec, secInfo := range netSecs {
			if sec == "lan" {
				secInfo.Up = backend.getIfaceStatus("lan")
			} else if sec == "planb" {
				secInfo.Up = backend.getIfaceStatus("planb")
			} else if sec == "wan" {
				secInfo.Up = backend.getIfaceStatus("wan")
			}
			l.Debugln("sec=", sec, "info=", secInfo.Device, secInfo.IPNet.IP.String(), net.IP(secInfo.IPNet.Mask).String(), secInfo.Gateway, secInfo.Proto, secInfo.Up)
		}
	}
}
