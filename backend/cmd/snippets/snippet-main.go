package main

import (
	"log"
	"net"
	"time"
)

func main() {
	//uciMain()
	c, err := net.DialTimeout("tcp", "www.baidu.com:443", time.Second*3)
	if err == nil {
		c.Close()
	}
	log.Println("dial err=", err)
	c, err = net.DialTimeout("tcp", "www.qq.com:443", time.Second*3)
	if err == nil {
		c.Close()
	}
	log.Println("dial err=", err)
}
