package service

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
	"golang.org/x/sys/unix"
)

func capacityLiveBase(t *testing.T) string {
	t.Helper()
	base := os.Getenv("QUICKSTART_MULTI_LAN_CAPACITY_BASE")
	if base == "" {
		t.Skip("requires explicit M10 isolated capacity harness")
	}
	switch flag.Lookup("test.run").Value.String() {
	case "^TestMultiLANCapacityBackend$", "^TestMultiLANCapacityPackets$", "^TestMultiLANCapacityMeasure$", "^TestMultiLANFullApplication$":
	default:
		t.Fatal("capacity fixtures require an exclusive test entrypoint")
	}
	marker, err := os.ReadFile(filepath.Join(base, "owner"))
	current, _ := os.Readlink("/proc/self/ns/net")
	initial, _ := os.Readlink("/proc/1/ns/net")
	if err != nil || string(marker) != "M10 isolated capacity" || !strings.HasPrefix(base, "/tmp/quickstart-multi-lan/m10-") || current == initial {
		t.Fatal("unsafe capacity fixture")
	}
	return base
}

// Inject the same 256 source identities in every isolated LAN. This exercises
// real BPF maps and forwarding without creating hundreds of host interfaces.
func TestMultiLANCapacityPackets(t *testing.T) {
	base := capacityLiveBase(t)
	iface, err := net.InterfaceByName(os.Getenv("QUICKSTART_MULTI_LAN_CAPACITY_IFACE"))
	if err != nil {
		t.Fatal(err)
	}
	destination, err := net.ParseMAC(os.Getenv("QUICKSTART_MULTI_LAN_CAPACITY_ROUTER_MAC"))
	if err != nil {
		t.Fatal(err)
	}
	subnet, err := strconv.Atoi(os.Getenv("QUICKSTART_MULTI_LAN_CAPACITY_SUBNET"))
	if err != nil || subnet < 201 || subnet > 204 {
		t.Fatal("invalid capacity prefix")
	}
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int((unix.ETH_P_IP&0xff)<<8|unix.ETH_P_IP>>8))
	if err != nil {
		t.Fatal(err)
	}
	defer unix.Close(fd)
	target := &unix.SockaddrLinklayer{Ifindex: iface.Index, Halen: 6}
	copy(target.Addr[:], destination)
	frame := make([]byte, 14+20+8+96*(subnet-200))
	copy(frame[:6], destination)
	frame[12], frame[13] = 8, 0
	ip := frame[14:34]
	ip[0] = 0x45
	binary.BigEndian.PutUint16(ip[2:4], uint16(len(frame)-14))
	ip[8], ip[9] = 64, 17
	copy(ip[16:20], []byte{203, 0, 113, 2})
	udp := frame[34:42]
	binary.BigEndian.PutUint16(udp[:2], 40000)
	binary.BigEndian.PutUint16(udp[2:4], 7777)
	binary.BigEndian.PutUint16(udp[4:6], uint16(len(frame)-34))
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		for i := 1; i <= 256; i++ {
			copy(frame[6:12], []byte{2, 0, 0, 0, byte(i >> 8), byte(i)})
			copy(ip[12:16], []byte{10, byte(subnet), byte(i / 250), byte(i%250 + 2)})
			ip[10], ip[11] = 0, 0
			sum := uint32(0)
			for j := 0; j < 20; j += 2 {
				sum += uint32(binary.BigEndian.Uint16(ip[j : j+2]))
			}
			for sum>>16 != 0 {
				sum = (sum & 65535) + (sum >> 16)
			}
			binary.BigEndian.PutUint16(ip[10:12], ^uint16(sum))
			if err := unix.Sendto(fd, frame, 0, target); err != nil {
				t.Fatal(err)
			}
		}
		select {
		case <-ticker.C:
		case <-time.After(2 * time.Second):
			t.Fatal("packet injection stalled")
		}
		if _, err := os.Stat(filepath.Join(base, "stop")); err == nil {
			return
		}
	}
}

func TestMultiLANCapacityBackend(t *testing.T) {
	base := capacityLiveBase(t)
	var scopes []*models.LANScope
	raw, err := os.ReadFile(filepath.Join(base, "scopes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &scopes); err != nil || len(scopes) < 1 || len(scopes) > 4 {
		t.Fatal("invalid scope fixture", err)
	}
	resolver := &LANScopeModule{reader: fakeLANScopeReader{scopes: scopes}, now: time.Now}
	inventory := newDeviceInventoryModuleForTest(systemDeviceInventorySource{scopes: resolver}, filepath.Join(base, "history.json"), "m10-capacity", time.Now, 2048)
	inventory.cacheTTL = defaultDeviceInventoryCacheTTL
	native := NewNativeRateLimitProvider("http://127.0.0.1:1", nil).(*nativeRateLimitProvider)
	native.registry = &nativeEngineRegistry{configDir: filepath.Join(base, "config")}
	native.offload = nil
	traffic := newDeviceTrafficModuleForTest(inventory, scopedDeviceTrafficSampler{native: native}, time.Now)
	deviceRestrictionConfigDir = func() string { return filepath.Join(base, "config") }
	deviceAccessRestrictionApply = func(ctx context.Context) error {
		out, err := exec.CommandContext(ctx, "sh", filepath.Join(base, "load-firewall")).CombinedOutput()
		if err != nil {
			return fmt.Errorf("isolated firewall: %w: %s", err, out)
		}
		return nil
	}
	deviceRestrictionApply = func(context.Context, []string) error { return nil }
	policy := newDevicePolicyModuleForTest(&systemDevicePolicyStore{inventory: inventory})
	var writeMu sync.Mutex
	writes := map[string]int{}
	writeTimes := map[string][]time.Time{}
	historyStore := newTrafficInsightsFileStore(filepath.Join(base, "traffic.json"))
	historyStore.persist = func(path string, raw []byte) error {
		if err := persistClassificationOverrides(path, raw); err != nil {
			return err
		}
		writeMu.Lock()
		writes[path]++
		writeTimes[path] = append(writeTimes[path], time.Now())
		writeMu.Unlock()
		return nil
	}
	insights := NewTrafficInsightsModule(historyStore, policy)
	for _, scope := range scopes {
		child := insights.forContext(withLANScope(context.Background(), scope))
		if err := child.load(); err != nil {
			t.Fatal(err)
		}
		for i := 1; i <= 256; i++ {
			id := fmt.Sprintf("mac:02:00:00:00:%02x:%02x", i>>8, i&255)
			child.document.Quotas[id] = &models.TrafficQuota{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 1 << 50, Action: "notify"}
		}
		if err := child.persist(time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	insights.AttachLANScopes(resolver)
	insights.AttachCollector(traffic.Snapshot)
	groupStore := newJSONDeviceGroupStore(filepath.Join(base, "groups.json"))
	schedules := []*models.GroupSchedule{}
	for hour := 0; hour < 24; hour++ {
		schedules = append(schedules, &models.GroupSchedule{ID: fmt.Sprintf("hour-%d", hour), Enabled: true, Days: []int{0, 1, 2, 3, 4, 5, 6}, StartMinute: hour*60 + 10, EndMinute: hour*60 + 11, Action: "block"})
	}
	if _, err := groupStore.Mutate(withLANScope(context.Background(), scopes[0]), &models.DeviceGroupMutationRequest{Action: "upsert_group", Group: &models.DeviceGroup{ID: "hourly", Name: "hourly isolated check", Members: []string{"mac:02:00:00:00:00:01"}, Policy: &models.GroupPolicy{Schedules: schedules}}}, ""); err != nil {
		t.Fatal(err)
	}
	groups := NewDeviceGroupModule(groupStore, &defaultGroupPolicyApplier{policy: policy})
	groups.inventory = inventory
	groups.AttachLANScopes(resolver)
	backend := &ServiceBackend{lanScopes: resolver, deviceInventory: inventory, deviceTraffic: traffic, trafficInsights: insights, deviceGroups: groups, networkAudit: NewNetworkAuditModule(filepath.Join(base, "audit.json"), nil)}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "GET" {
			http.Error(w, "read only capacity endpoint", 405)
			return
		}
		if r.URL.Path == "/metrics" {
			writeMu.Lock()
			copyWrites := map[string]int{}
			lastHourWrites := map[string]int{}
			for name, count := range writes {
				copyWrites[name] = count
				kept := writeTimes[name][:0]
				for _, stamp := range writeTimes[name] {
					if time.Since(stamp) < time.Hour {
						kept = append(kept, stamp)
					}
				}
				writeTimes[name] = kept
				lastHourWrites[name] = len(kept)
			}
			writeMu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"writes": copyWrites, "writesLastHour": lastHourWrites, "at": time.Now().UTC()})
			return
		}
		ctx, err := backend.PrepareLANRequest(r.Context(), r)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var response any
		switch r.URL.Path {
		case "/cgi-bin/luci/istore/lanctrl/v2/devices/":
			response, err = backend.GetDeviceInventoryV2(ctx)
		case "/cgi-bin/luci/istore/lanctrl/v2/device-traffic/":
			response, err = backend.GetDeviceTrafficV2(ctx)
		case "/cgi-bin/luci/istore/lanctrl/v2/networks/":
			response, err = backend.GetLANNetworks(ctx)
		default:
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	address := "127.0.0.1:19481"
	readyFile := "backend.ready"
	if os.Getenv("QUICKSTART_MULTI_LAN_FULL_APPLICATION") == "1" {
		address = "127.0.0.1:19483"
		readyFile = "core.ready"
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener)
	defer server.Shutdown(context.Background())
	if err := os.WriteFile(filepath.Join(base, readyFile), []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		t.Fatal(err)
	}
	for {
		time.Sleep(time.Second)
		if _, err := os.Stat(filepath.Join(base, "stop")); err == nil {
			return
		}
	}
}

func TestMultiLANCapacityMeasure(t *testing.T) {
	base := capacityLiveBase(t)
	type process struct {
		PID      int   `json:"pid"`
		RSSKiB   int64 `json:"rssKiB"`
		Ticks    int64 `json:"ticks"`
		FDs      int   `json:"fds"`
		BPFBytes int64 `json:"bpfBytes"`
		BPFMaps  int   `json:"bpfMaps"`
	}
	processes := map[string]process{}
	files, _ := filepath.Glob(filepath.Join(base, "engine-*.pid"))
	files = append(files, filepath.Join(base, "backend.pid"))
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
		prefix := fmt.Sprintf("/proc/%d", pid)
		status, err := os.ReadFile(filepath.Join(prefix, "status"))
		if err != nil {
			t.Fatal(err)
		}
		value := process{PID: pid}
		for _, line := range strings.Split(string(status), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				value.RSSKiB, _ = strconv.ParseInt(strings.Fields(line)[1], 10, 64)
			}
		}
		stat, err := os.ReadFile(filepath.Join(prefix, "stat"))
		if err != nil {
			t.Fatal(err)
		}
		fields := strings.Fields(string(stat)[strings.LastIndex(string(stat), ")")+2:])
		user, _ := strconv.ParseInt(fields[11], 10, 64)
		system, _ := strconv.ParseInt(fields[12], 10, 64)
		childUser, _ := strconv.ParseInt(fields[13], 10, 64)
		childSystem, _ := strconv.ParseInt(fields[14], 10, 64)
		value.Ticks = user + system + childUser + childSystem
		fds, _ := os.ReadDir(filepath.Join(prefix, "fd"))
		value.FDs = len(fds)
		infos, _ := filepath.Glob(filepath.Join(prefix, "fdinfo", "*"))
		seen := map[string]bool{}
		for _, info := range infos {
			raw, _ := os.ReadFile(info)
			id := ""
			memory := int64(0)
			for _, line := range strings.Split(string(raw), "\n") {
				parts := strings.Fields(line)
				if len(parts) != 2 {
					continue
				}
				if parts[0] == "map_id:" {
					id = parts[1]
				}
				if parts[0] == "memlock:" {
					memory, _ = strconv.ParseInt(parts[1], 10, 64)
				}
			}
			if id != "" && !seen[id] {
				seen[id] = true
				value.BPFMaps++
				value.BPFBytes += memory
			}
		}
		processes[filepath.Base(file)] = value
	}
	var cpuAt unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &cpuAt); err != nil {
		t.Fatal(err)
	}
	api := []map[string]any{}
	iterations := 1
	if os.Getenv("QUICKSTART_MULTI_LAN_CAPACITY_LATENCY") == "1" {
		iterations = 100
	}
	client := &http.Client{Timeout: 15 * time.Second}
	var scopes []*models.LANScope
	raw, _ := os.ReadFile(filepath.Join(base, "scopes.json"))
	if err := json.Unmarshal(raw, &scopes); err != nil {
		t.Fatal(err)
	}
	for _, scope := range scopes {
		for _, endpoint := range []string{"devices", "device-traffic"} {
			durations := []float64{}
			count := 0
			activeDevices := 0
			totalUpload, totalDownload := int64(0), int64(0)
			state := ""
			maxAge := float64(0)
			for i := 0; i < iterations; i++ {
				start := time.Now()
				request, _ := http.NewRequest("GET", "http://127.0.0.1:19481/cgi-bin/luci/istore/lanctrl/v2/"+endpoint+"/?network="+scope.NetworkID, nil)
				if _, err := os.Stat(filepath.Join(base, "full-application")); err == nil {
					request.Header.Set("X-Forwarded-Sid", capacitySession)
					request.Header.Set("X-Forwarded-Token", capacityToken)
				}
				response, err := client.Do(request)
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(response.Body)
				response.Body.Close()
				if err != nil || response.StatusCode != 200 {
					t.Fatalf("API %d: %s %v", response.StatusCode, body, err)
				}
				durations = append(durations, float64(time.Since(start).Microseconds())/1000)
				var payload struct {
					Result struct {
						Devices []struct {
							NetworkID string `json:"networkId"`
						} `json:"devices"`
						Items []struct {
							NetworkID     string `json:"networkId"`
							State         string `json:"state"`
							SampledAt     string `json:"sampledAt"`
							UploadBytes   int64  `json:"uploadBytes"`
							DownloadBytes int64  `json:"downloadBytes"`
						} `json:"items"`
						Health struct {
							State string `json:"state"`
						} `json:"health"`
					} `json:"result"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatal(err)
				}
				state = payload.Result.Health.State
				if endpoint == "devices" {
					count = len(payload.Result.Devices)
					for _, device := range payload.Result.Devices {
						if device.NetworkID != scope.NetworkID {
							t.Fatal("inventory escaped scope")
						}
					}
				} else {
					count = len(payload.Result.Items)
					activeDevices = 0
					totalUpload, totalDownload = 0, 0
					for _, item := range payload.Result.Items {
						totalUpload += item.UploadBytes
						totalDownload += item.DownloadBytes
						if item.UploadBytes+item.DownloadBytes > 0 {
							activeDevices++
						}
						if item.NetworkID != scope.NetworkID {
							t.Fatal("traffic escaped scope")
						}
						if item.State != "ready" {
							state = item.State
						}
						if stamp, err := time.Parse(time.RFC3339Nano, item.SampledAt); err == nil {
							age := time.Since(stamp).Seconds()
							if age > maxAge {
								maxAge = age
							}
						}
					}
				}
			}
			sort.Float64s(durations)
			api = append(api, map[string]any{"networkId": scope.NetworkID, "endpoint": endpoint, "devices": count, "activeDevices": activeDevices, "uploadBytes": totalUpload, "downloadBytes": totalDownload, "state": state, "maxSampleAgeSeconds": maxAge, "samples": iterations, "p95Ms": durations[(len(durations)-1)*95/100]})
		}
	}
	metricsURL := "http://127.0.0.1:19481/metrics"
	if _, err := os.Stat(filepath.Join(base, "full-application")); err == nil {
		metricsURL = "http://127.0.0.1:19484/metrics"
	}
	response, err := client.Get(metricsURL)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	var metrics any
	if err := json.Unmarshal(raw, &metrics); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(map[string]any{"cpuSampleMonotonicSeconds": float64(cpuAt.Sec) + float64(cpuAt.Nsec)/1e9, "cpuIncludesChildren": true, "at": time.Now().UTC(), "processes": processes, "api": api, "metrics": metrics})
	fmt.Printf("CAPACITY=%s\n", encoded)
}
