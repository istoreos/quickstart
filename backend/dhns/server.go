package dhns

import (
	"container/list"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/linkease/quick-start/istore-backend/message"
	uuid "github.com/satori/go.uuid"
)

var (
	errAlreadyDie = errors.New("already die")
)

type ConnItem struct {
	id       string
	lastTime time.Time
	conn     chan net.Conn
	el       *list.Element
}

type ConnItemObj struct {
	id   string
	conn net.Conn
}

type DhnsServer struct {
	mu         sync.Mutex
	client     *http.Client
	lastConn   net.Conn
	pongCh     chan struct{}
	writeMsgCh chan *message.Message

	//conn bus
	connBusMap    map[string]*ConnItem
	connBusRegist chan *ConnItem
	connBusNotify chan *ConnItemObj

	dieOnce sync.Once
	dieCh   chan struct{}
}

func NewDhnsServer() *DhnsServer {
	srv := &DhnsServer{
		connBusMap:    make(map[string]*ConnItem),
		connBusRegist: make(chan *ConnItem, 8),
		connBusNotify: make(chan *ConnItemObj, 8),

		pongCh:     make(chan struct{}, 1),
		writeMsgCh: make(chan *message.Message, 8),
		dieCh:      make(chan struct{}),
	}
	srv.client = &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return srv.DialDhns()
			},
		},
	}
	go srv.connBus()

	return srv
}

func (srv *DhnsServer) connBus() {
	l := list.New()
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		if err := srv.connBusOnce(l, ticker); err != nil {
			return
		}
	}
}

func (srv *DhnsServer) connBusOnce(l *list.List, ticker *time.Ticker) error {
	select {
	case <-srv.dieCh:
		return errAlreadyDie
	case <-ticker.C:
		now := time.Now().Add(time.Second * (-10))
		for {
			el := l.Front()
			if el == nil {
				break
			}
			item := el.Value.(*ConnItem)
			if item.lastTime.Before(now) {
				//timeout
				delete(srv.connBusMap, item.id)
				l.Remove(item.el)
				close(item.conn)
			} else {
				//not timeout
				break
			}
		}
	case item := <-srv.connBusRegist:
		if _, ok := srv.connBusMap[item.id]; !ok {
			srv.connBusMap[item.id] = item
			item.lastTime = time.Now()
			item.el = l.PushBack(item)
		} else {
			close(item.conn)
		}
	case obj := <-srv.connBusNotify:
		if item, ok := srv.connBusMap[obj.id]; ok {
			delete(srv.connBusMap, item.id)
			if item.el != nil {
				l.Remove(item.el)
				item.el = nil
			}
			if item.conn != nil {
				select {
				case <-srv.dieCh:
					obj.conn.Close()
				case item.conn <- obj.conn:
				}
			}
			close(item.conn)
		} else {
			obj.conn.Close()
		}
	}
	return nil

}

func (srv *DhnsServer) connItemRegist(item *ConnItem) error {
	select {
	case <-srv.dieCh:
		return errAlreadyDie
	case srv.connBusRegist <- item:
		return nil
	}
}

func (srv *DhnsServer) connItemNotify(obj *ConnItemObj) error {
	select {
	case <-srv.dieCh:
		obj.conn.Close()
		return errAlreadyDie
	case srv.connBusNotify <- obj:
		return nil
	}
}

func newV4UUID() string {
	return uuid.NewV4().String()
}

func (srv *DhnsServer) Client() *http.Client {
	return srv.client
}

func (srv *DhnsServer) DialDhns() (net.Conn, error) {
	msg := &message.MessageDhnsNewConn{
		ConnID: newV4UUID(),
	}
	item := &ConnItem{
		id:   msg.ConnID,
		conn: make(chan net.Conn, 1),
	}
	err := srv.connItemRegist(item)
	if err != nil {
		return nil, err
	}
	select {
	case srv.writeMsgCh <- &message.Message{
		Type: message.MsgTypeDhnsNewConn,
		Msg:  msg,
	}:
	case <-srv.dieCh:
		return nil, errAlreadyDie
	}
	select {
	case c, ok := <-item.conn:
		if ok {
			//l.Debugln("got conn id=", item.id)
			return c, nil
		}
	case <-srv.dieCh:
	}
	return nil, io.ErrUnexpectedEOF
}

func (srv *DhnsServer) PutDhnsConn(id string, c net.Conn) error {
	return srv.connItemNotify(&ConnItemObj{
		id:   id,
		conn: c,
	})
}

func (srv *DhnsServer) HandleConn(c net.Conn) {
	var hasConn bool
	srv.mu.Lock()
	if srv.lastConn == nil {
		srv.lastConn = c
	} else {
		hasConn = true
	}
	srv.mu.Unlock()
	if hasConn {
		c.Close()
		return
	}
	defer srv.closeConn(c)
	log.Println("new dhns connect in")

	go srv.dhnsWrite(c)
	data := make([]byte, 4096)
	for {
		c.SetReadDeadline(time.Now().Add(time.Second * 60))
		_, dataType, err := message.ReadMessage(c, data)
		if err != nil {
			log.Println("dhns server read err=", err)
			return
		}
		switch dataType {
		case message.MsgTypePing:
			//log.Println("got ping")
			select {
			case srv.pongCh <- struct{}{}:
			case <-srv.dieCh:
			}
		default:
		}
	}
}

func (srv *DhnsServer) dhnsWrite(c net.Conn) {
	msg := &message.MessagePingPong{
		Msg: "pong",
	}
	for {
		select {
		case <-srv.pongCh:
			c.SetWriteDeadline(time.Now().Add(time.Second * 5))
			if err := message.WriteMessage(c, message.MsgTypePong, msg); err != nil {
				srv.closeConn(c)
				return
			}
		case msg := <-srv.writeMsgCh:
			c.SetWriteDeadline(time.Now().Add(time.Second * 5))
			if err := message.WriteMessage(c, uint16(msg.Type), msg.Msg); err != nil {
				srv.closeConn(c)
				return
			}
		case <-srv.dieCh:
			srv.closeConn(c)
			return
		}
	}
}

func (srv *DhnsServer) closeConn(c net.Conn) {
	var lastConn net.Conn
	srv.mu.Lock()
	if srv.lastConn == c {
		lastConn = srv.lastConn
		srv.lastConn = nil
	}
	srv.mu.Unlock()
	if lastConn != nil {
		log.Println("dhns server conn closed")
		lastConn.Close()
	}
}

func (srv *DhnsServer) Close() error {
	srv.dieOnce.Do(func() {
		close(srv.dieCh)
	})
	return nil
}
