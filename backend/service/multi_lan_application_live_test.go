package service

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
	"golang.org/x/sys/unix"
)

// Synthetic local rpcd session, valid only inside this disposable rootfs.
const capacitySession = "11111111111111111111111111111111"
const capacityToken = "m10-isolated"

// Run the unmodified production executable with private filesystem, process,
// mount and network views. Read-only firmware tools are shared. Runtime/session
// RPC fixtures supply netifd/rpcd metadata; real neighbors, BPF and packets are
// the owned namespace's kernel state. No host ubus socket is exposed.
func TestMultiLANFullApplication(t *testing.T) {
	base := capacityLiveBase(t)
	root := filepath.Join(base, "rootfs")
	if os.Getenv("QUICKSTART_MULTI_LAN_APPLICATION_CHILD") == "1" {
		runtime.LockOSThread()
		if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
			t.Fatal(err)
		}
		for _, dir := range []string{"bin", "sbin", "usr", "lib", "dev", "sys"} {
			target := filepath.Join(root, dir)
			if err := os.MkdirAll(target, 0755); err != nil {
				t.Fatal(err)
			}
			if err := unix.Mount("/"+dir, target, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
				t.Fatal(err)
			}
			if err := unix.Mount("", target, "", unix.MS_BIND|unix.MS_REMOUNT|unix.MS_RDONLY, ""); err != nil {
				t.Fatal(err)
			}
		}
		var scopes []*models.LANScope
		raw, err := os.ReadFile(filepath.Join(base, "scopes.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &scopes); err != nil {
			t.Fatal(err)
		}
		for index := range scopes {
			source := filepath.Join(base, fmt.Sprintf("engine-%d", index+1))
			target := filepath.Join(root, strings.TrimPrefix(source, "/"))
			if err := unix.Mount(source, target, "", unix.MS_BIND, ""); err != nil {
				t.Fatal(err)
			}
			if err := unix.Mount("", target, "", unix.MS_BIND|unix.MS_REMOUNT|unix.MS_RDONLY, ""); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.MkdirAll(filepath.Join(root, "proc"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := unix.Mount("proc", filepath.Join(root, "proc"), "proc", unix.MS_NOSUID|unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
			t.Fatal(err)
		}
		if err := unix.Chroot(root); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir("/"); err != nil {
			t.Fatal(err)
		}
		environment := []string{}
		for _, value := range os.Environ() {
			if !strings.HasPrefix(value, "PATH=") {
				environment = append(environment, value)
			}
		}
		environment = append(environment, "PATH=/opt/test-bin:/usr/sbin:/usr/bin:/sbin:/bin")
		if err := syscall.Exec("/opt/quickstart", []string{"quickstart", "serve", "--addr", "127.0.0.1:19481"}, environment); err != nil {
			t.Fatal(err)
		}
		return
	}
	if os.Getenv("QUICKSTART_MULTI_LAN_APPLICATION_REUSE_ROOT") != "1" {
		if err := prepareCapacityApplication(base, root); err != nil {
			t.Fatal(err)
		}
	}
	metadata, closeMetadata, err := capacityApplicationWrites(root)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMetadata()
	server := &http.Server{Addr: "127.0.0.1:19484", Handler: metadata, ReadHeaderTimeout: 5 * time.Second}
	go server.ListenAndServe()
	defer server.Shutdown(context.Background())
	command := exec.Command(os.Args[0], "-test.run", "^TestMultiLANFullApplication$", "-test.timeout", "26h")
	command.Env = append(os.Environ(), "QUICKSTART_MULTI_LAN_APPLICATION_CHILD=1")
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{Cloneflags: unix.CLONE_NEWNS | unix.CLONE_NEWPID, Pdeathsig: unix.SIGKILL}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "backend.pid"), []byte(fmt.Sprint(command.Process.Pid)), 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(time.Minute)
	lastResponse := ""
	legacy := os.Getenv("QUICKSTART_MULTI_LAN_APPLICATION_LEGACY") == "1"
	endpoint := "networks"
	if legacy {
		endpoint = "devices"
	}
	for {
		select {
		case err := <-done:
			t.Fatal("production application exited", err)
		default:
		}
		request, _ := http.NewRequest("GET", "http://127.0.0.1:19481/cgi-bin/luci/istore/lanctrl/v2/"+endpoint+"/", nil)
		request.Header.Set("X-Forwarded-Sid", capacitySession)
		request.Header.Set("X-Forwarded-Token", capacityToken)
		response, err := client.Do(request)
		if err == nil {
			var value struct {
				Result *struct {
					Networks []json.RawMessage `json:"networks"`
					Devices  []json.RawMessage `json:"devices"`
				} `json:"result"`
			}
			err = json.NewDecoder(response.Body).Decode(&value)
			response.Body.Close()
			encoded, _ := json.Marshal(value)
			lastResponse = string(encoded)
			if err == nil && value.Result != nil && (len(value.Result.Networks) > 0 || legacy && len(value.Result.Devices) > 0) {
				break
			}
		}
		if time.Now().After(deadline) {
			command.Process.Kill()
			t.Fatal("production application not ready", err, lastResponse)
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(base, "backend.ready"), []byte("production executable"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal("production application exited", err)
	}
}

func prepareCapacityApplication(base, root string) error {
	for _, dir := range []string{"etc/config", "etc/quickstart/networks", "opt/test-bin", "tmp/run", "tmp/lock", "tmp/.uci", "proc", "root", "data"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			return err
		}
	}
	if err := os.Symlink("/tmp", filepath.Join(root, "var")); err != nil && !os.IsExist(err) {
		return err
	}
	copyFile := func(source, target string) error {
		raw, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0600)
	}
	for _, name := range []string{"openwrt_release", "os-release", "TZ", "localtime", "board.json", "passwd", "group"} {
		if _, err := os.Stat("/etc/" + name); err == nil {
			if err := copyFile("/etc/"+name, filepath.Join(root, "etc", name)); err != nil {
				return err
			}
		}
	}
	if err := copyFile(filepath.Join(base, "application"), filepath.Join(root, "opt/quickstart")); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Join(root, "opt/quickstart"), 0700); err != nil {
		return err
	}
	var scopes []*models.LANScope
	raw, err := os.ReadFile(filepath.Join(base, "scopes.json"))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &scopes); err != nil {
		return err
	}
	network, firewall, engines := "", "config defaults 'main'\n option input 'ACCEPT'\n option output 'ACCEPT'\n option forward 'ACCEPT'\nconfig zone 'local'\n option name 'shared'\n option input 'ACCEPT'\n option output 'ACCEPT'\n option forward 'ACCEPT'\n", ""
	interfaces := []map[string]any{}
	for index, scope := range scopes {
		network += fmt.Sprintf("config interface '%s'\n option proto 'static'\n option device '%s'\n", scope.NetworkID, scope.Device)
		firewall += fmt.Sprintf(" list network '%s'\n list device '%s'\n", scope.NetworkID, scope.Device)
		engines += fmt.Sprintf("config quickstart-netpolicy '%s'\n option enabled '1'\n option network '%s'\n option interface '%s'\n option port '%d'\n option data_dir '%s'\n", scope.NetworkID, scope.NetworkID, scope.Device, 19501+index, filepath.Join(base, fmt.Sprintf("engine-%d", index+1)))
		if err := os.MkdirAll(filepath.Join(root, strings.TrimPrefix(filepath.Join(base, fmt.Sprintf("engine-%d", index+1)), "/")), 0700); err != nil {
			return err
		}
		ipv4 := []map[string]any{}
		ipv6 := []map[string]any{}
		for _, prefix := range scope.IPv4Prefixes {
			parts := strings.Split(prefix, "/")
			ipv4 = append(ipv4, map[string]any{"address": parts[0], "mask": 23})
		}
		for _, prefix := range scope.IPv6Prefixes {
			parts := strings.Split(prefix, "/")
			ipv6 = append(ipv6, map[string]any{"address": parts[0], "mask": 64})
		}
		interfaces = append(interfaces, map[string]any{"interface": scope.NetworkID, "l3_device": scope.Device, "device": scope.Device, "proto": "static", "up": true, "ipv4-address": ipv4, "ipv6-address": ipv6})
		dir := filepath.Join(root, "etc/quickstart/networks", scope.NetworkID)
		if scope.NetworkID == defaultLAN {
			dir = filepath.Join(root, "etc/quickstart")
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		if err := copyFile(lanStatePath(filepath.Join(base, "traffic.json"), scope.NetworkID), filepath.Join(dir, "traffic-insights.json")); err != nil {
			return err
		}
	}
	runtimeJSON, err := json.Marshal(map[string]any{"interface": interfaces})
	if err != nil {
		return err
	}
	files := map[string][]byte{"etc/config/network": []byte(network), "etc/config/firewall": []byte(firewall), "etc/config/quickstart-netpolicy": []byte(engines), "etc/config/dhcp": []byte("config dnsmasq 'main'\n"), "etc/config/quickstart": []byte("config quickstart 'main'\n option disable_dhns '1'\n"), "opt/runtime.json": runtimeJSON, "tmp/dhcp.leases": {}}
	for name, value := range files {
		if err := os.WriteFile(filepath.Join(root, name), value, 0600); err != nil {
			return err
		}
	}
	shim := `#!/bin/sh
while [ "$#" -gt 0 ] && [ "$1" != call ]; do shift; done
[ "$#" -ge 3 ] || exit 1
case "$2:$3" in
 network.interface:dump) cat /opt/runtime.json;;
 session:get) printf '{"values":{"username":"root","token":"m10-isolated"}}';;
 luci-rpc:getHostHints) printf '{}';;
 network.wireless:status|luci:getWirelessDevices) printf '{}';;
 service:list) printf '{}';;
 *) exit 1;;
esac
`
	return os.WriteFile(filepath.Join(root, "opt/test-bin/ubus"), []byte(shim), 0700)
}

// Count atomic history replacements from the production process, including
// short critical checkpoints. Sampling file mtimes could miss multiple writes.
func capacityApplicationWrites(root string) (http.Handler, func(), error) {
	fd, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC)
	if err != nil {
		return nil, nil, err
	}
	dirs, err := filepath.Glob(filepath.Join(root, "etc/quickstart/networks/*"))
	if err != nil {
		unix.Close(fd)
		return nil, nil, err
	}
	names := map[int32]string{}
	dirs = append(dirs, filepath.Join(root, "etc/quickstart"))
	for _, dir := range dirs {
		watch, err := unix.InotifyAddWatch(fd, dir, unix.IN_MOVED_TO)
		if err != nil {
			unix.Close(fd)
			return nil, nil, err
		}
		names[int32(watch)] = filepath.Base(dir)
		if dir == filepath.Join(root, "etc/quickstart") {
			names[int32(watch)] = defaultLAN
		}
	}
	var mu sync.Mutex
	events := map[string][]time.Time{}
	counts := map[string]int{}
	stop := make(chan struct{})
	go func() {
		buffer := make([]byte, 16384)
		for {
			select {
			case <-stop:
				return
			default:
			}
			n, err := unix.Read(fd, buffer)
			if err == unix.EAGAIN {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if err != nil {
				return
			}
			for offset := 0; offset+unix.SizeofInotifyEvent <= n; {
				watch := int32(binary.LittleEndian.Uint32(buffer[offset:]))
				size := int(binary.LittleEndian.Uint32(buffer[offset+12:]))
				end := offset + unix.SizeofInotifyEvent + size
				if end > n {
					break
				}
				name := strings.TrimRight(string(buffer[offset+unix.SizeofInotifyEvent:end]), "\x00")
				if name == "traffic-insights.json" {
					mu.Lock()
					network := names[watch]
					counts[network]++
					events[network] = append(events[network], time.Now())
					mu.Unlock()
				}
				offset = end
			}
		}
	}()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		recent := map[string]int{}
		for name, times := range events {
			kept := times[:0]
			for _, at := range times {
				if time.Since(at) < time.Hour {
					kept = append(kept, at)
				}
			}
			events[name] = kept
			recent[name] = len(kept)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"writes": counts, "writesLastHour": recent, "application": "production executable", "at": time.Now().UTC()})
	})
	return handler, func() { close(stop); unix.Close(fd) }, nil
}
