package dhns

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/linkease/quick-start/istore-backend/utils"
)

func (cli *DhnsClient) serveStaticIP(w http.ResponseWriter, r *http.Request) {
	var staticNet DhnsStatic
	err := json.NewDecoder(r.Body).Decode(&staticNet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	maskIP := net.ParseIP(staticNet.Mask)
	if maskIP == nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	prefixSize, _ := net.IPMask(maskIP.To4()).Size()
	cmdList := []string{
		"ip link set dheth1 up",
	}
	if !staticNet.NetworkCheck {
		cmdList = append(cmdList, "ip addr flush dev dheth1")
	}
	cmdList = append(cmdList, fmt.Sprintf("ip addr add %s/%d dev dheth1", staticNet.IP, prefixSize))
	if staticNet.Gateway != "" {
		cmdList = append(cmdList, fmt.Sprintf("ip route add default via %s", staticNet.Gateway))
	}
	err = utils.BatchRun(r.Context(), cmdList, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if staticNet.NetworkCheck {
		var failed bool
		if !isNetworkOK() {
			w.Write([]byte("FAILED"))
			failed = true
		}
		utils.BatchRun(r.Context(), []string{
			fmt.Sprintf("ip addr del %s/%d dev dheth1", staticNet.IP, prefixSize),
		}, 5)
		if failed {
			return
		}
	}
	w.Write([]byte("OK"))
}

func (cli *DhnsClient) serveCheckAlive(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func closeConn(c net.Conn) {
	if c != nil {
		c.Close()
	}
}

func isNetworkOK() bool {
	c, err := net.DialTimeout("tcp", "114.114.114.114:53", time.Second*3)
	closeConn(c)
	if err != nil {
		// check again
		c, err = net.DialTimeout("tcp", "223.5.5.5:53", time.Second*3)
		closeConn(c)
	}
	if err != nil {
		return false
	}
	return true
}
