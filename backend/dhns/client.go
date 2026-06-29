package dhns

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/message"
)

type pipeAddr struct {
}

func (pa *pipeAddr) Network() string {
	return "pipe"
}

func (pa *pipeAddr) String() string {
	return "pipe"
}

type DhnsClient struct {
	unixSocket string
	lis        *MuxListener
	server     *http.Server
	dieOnce    sync.Once
	dieCh      chan struct{}
}

type DhnsStatic struct {
	IP           string `json:"ip"`
	Mask         string `json:"mask"`
	Gateway      string `json:"gateway"`
	NetworkCheck bool   `json:"networkCheck"`
}

func NewDhnsClient(unixSocket string) *DhnsClient {
	cli := &DhnsClient{
		unixSocket: unixSocket,
		lis:        NewMuxListener(&pipeAddr{}),
		dieCh:      make(chan struct{}),
	}
	router := httprouter.New()
	router.POST("/api/dhns/static/", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		cli.serveStaticIP(w, r)
	})
	router.GET("/api/dhns/alive/", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		cli.serveCheckAlive(w, r)
	})
	cli.server = &http.Server{
		Handler: router,
	}
	go cli.runConnect()
	go cli.server.Serve(cli.lis)
	go cli.forwardToRemote(80, 80)
	go cli.forwardToRemote(22, 22)
	return cli
}

func (cli *DhnsClient) Close() error {
	cli.dieOnce.Do(func() {
		close(cli.dieCh)
		cli.lis.Close()
		cli.server.Close()
	})
	return nil
}

func (cli *DhnsClient) runConnect() {
	for {
		select {
		case <-cli.dieCh:
			return
		default:
		}
		c, err := cli.connect("/api/dhns/connect/", "Proxyid", "")
		if err != nil {
			log.Println("connect dhns failed, err=", err)
			time.Sleep(time.Second * 5)
			continue
		}
		go cli.connWrite(c)
		cli.connRead(c)
		log.Println("dhns closed")
		c.Close()
		time.Sleep(time.Second * 5)
	}
}

func (cli *DhnsClient) connect(p, header, key string) (net.Conn, error) {
	c, err := net.Dial("unix", cli.unixSocket)
	if err != nil {
		return nil, err
	}
	//log.Println("crossNet targetAddr=", targetAddr, "key=", key)
	defer func() {
		if c != nil {
			c.Close()
		}
	}()

	req, _ := http.NewRequest("GET", p, nil)
	//req.Header.Set("Proxyid", key)
	req.Header.Set(header, key)
	req.Header.Set("Bufferlen", "000")
	b := &bytes.Buffer{}
	c.SetWriteDeadline(time.Now().Add(time.Second * 5))
	err = req.Write(b)
	if err != nil {
		return nil, err
	}
	data := b.Bytes()
	idx := bytes.Index(data, []byte("Bufferlen"))
	copy(data[idx:], []byte(fmt.Sprintf("BufferLen: %03d", len(data))))
	c.SetWriteDeadline(time.Now().Add(time.Second * 5))
	if n, err := c.Write(data); err != nil {
		return nil, err
	} else if n != len(data) {
		return nil, errors.New("partial write")
	}
	c.SetWriteDeadline(time.Time{})
	c2 := c
	c = nil
	return c2, nil
}

func (cli *DhnsClient) connWrite(c net.Conn) {
	ticker := time.NewTicker(time.Second * 20)
	defer ticker.Stop()

	msg := &message.Message{
		Type: message.MsgTypePing,
		Msg: &message.MessagePingPong{
			Msg: "ping",
		},
	}

	for {
		select {
		case <-cli.dieCh:
			// TODO maybe close twice here
			c.Close()
			return
		case <-ticker.C:
			c.SetWriteDeadline(time.Now().Add(time.Second * 5))
			err := message.WriteMessage(c, uint16(msg.Type), msg.Msg)
			//log.Println("dhns client write ping, err=", err)
			if err != nil {
				return
			}
		}
	}
}

func (cli *DhnsClient) connRead(c net.Conn) {
	data := make([]byte, 4096)
	for {
		c.SetReadDeadline(time.Now().Add(time.Second * 60))
		n, dataType, err := message.ReadMessage(c, data)
		if err != nil {
			log.Println("dhns client readMessage failed, err=", err)
			return
		}
		switch dataType {
		case message.MsgTypePong:
			//log.Println("got pong")
		case message.MsgTypeDhnsNewConn:
			var obj message.MessageDhnsNewConn
			err := json.Unmarshal(data[:n], &obj)
			if err != nil {
				log.Println("unmarshal err=", err)
				continue
			}
			go func() {
				newConn, err := cli.connect("/api/dhns/proxy/", "Proxyid", obj.ConnID)
				//log.Println("connect proxyID=", obj.ConnID, "err=", err)
				if err == nil {
					if err = cli.lis.PutConn(newConn); err != nil {
						newConn.Close()
					}
				}
			}()
		}
	}
}

func (cli *DhnsClient) forwardToRemote(lport, rport int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", lport))
	if err != nil {
		return
	}
	defer lis.Close()
	for {
		c, err := lis.Accept()
		if err != nil {
			return
		}
		go cli.forward(c, fmt.Sprintf("127.0.0.1:%d", rport))
	}
}

func (cli *DhnsClient) forward(local net.Conn, remoteAddr string) {
	defer local.Close()
	remote, err := cli.connect("/api/dhns/forward/", "TargetAddr", remoteAddr)
	if err != nil {
		log.Printf("remote dial failed: %v\n", err)
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
