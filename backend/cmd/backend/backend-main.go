package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/linkease/quick-start/istore-backend/dhns"
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/server"
	"github.com/linkease/quick-start/istore-backend/service"
	"github.com/urfave/cli/v2"
)

var (
	BuildVersion string
	BuildDate    string
)

func startMain(laddr, unixAddr string) error {
	l.Debugln("start serve at", laddr, "unix at", unixAddr)

	serviceBackend := service.NewServiceBackend()
	httpRouter := server.RouterInit(serviceBackend)
	unixRouter := server.UnixRouterInit(serviceBackend)

	lis, err := net.Listen("tcp", laddr)
	if err != nil {
		return err
	}
	defer lis.Close()

	if unixAddr != "" {
		go func() {
			if _, err := os.Stat(path.Dir(unixAddr)); err != nil && os.IsNotExist(err) {
				os.Mkdir(path.Dir(unixAddr), 0777)
			}
			os.Remove(unixAddr)
			lis, err := net.Listen("unix", unixAddr)
			if err != nil {
				l.Warnln("listen unix addr failed, err=", err)
				return
			}
			defer lis.Close()
			s := http.Server{
				Handler: unixRouter,
			}
			//s.SetKeepAlivesEnabled(false)
			s.Serve(lis)
		}()
	}

	s := http.Server{
		Handler: httpRouter,
	}
	s.SetKeepAlivesEnabled(false)
	s.Serve(lis)
	return nil
}

func main() {
	cliApp := &cli.App{
		Name:  "quickstart",
		Usage: "quickstart for iStoreOS.",
		Action: func(c *cli.Context) error {
			return mainPrompt()
		},
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "Show the current version",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "more",
						Value: false,
					},
				},
				Action: func(c *cli.Context) error {
					if c.Bool("more") {
						fmt.Println(service.VERSION, BuildVersion, BuildDate)
					} else {
						fmt.Println(service.VERSION)
					}
					return nil
				},
			},
			{
				Name: "serve",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "addr",
						Value: "127.0.0.1:3038",
					},
					&cli.StringFlag{
						Name:  "unix",
						Value: "",
					},
				},
				Action: func(c *cli.Context) error {
					return startMain(c.String("addr"), c.String("unix"))
				},
			},
			{
				Name:  "lan",
				Usage: "Change the LAN ip",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "ip",
						Usage: "set LAN ip, example: 192.168.100.1",
					},
					&cli.StringFlag{
						Name:  "mask",
						Value: "255.255.255.0",
						Usage: "set LAN mask, example: 255.255.255.0",
					},
				},
				Action: func(c *cli.Context) error {
					ip := c.String("ip")
					mask := c.String("mask")
					fmt.Println("ip=", ip, "mask=", mask)
					service.LanSetting(ip, mask, false)
					return nil
				},
			},
			{
				Name:  "showLanIP",
				Usage: "Show the LAN ip",
				Action: func(c *cli.Context) error {
					result, _ := service.GuideGetLanSetting(context.Background())
					fmt.Println("lanIp=", result.Result.LanIP)
					return nil
				},
			},
			{
				Name:  "blockChange",
				Usage: "disk add or remove then reload",
				Action: func(c *cli.Context) error {
					service.SmartReloadDisks()
					//在hotplug里面直接使用脚本处理
					// service.NasReloadDisk()
					return nil
				},
			},
			{
				Name:  "get-transparent-gateway",
				Usage: "Get Transparent Gateway",
				Action: func(c *cli.Context) error {
					ret, err := service.GuideGetTransparentGateway()
					if err != nil {
						fmt.Println(`{"result":"FAILED"}`)
					} else {
						data, _ := json.Marshal(ret)
						fmt.Println(string(data))
					}
					return nil
				},
			},
			{
				Name:  "set-transparent-gateway",
				Usage: "Setup Transparent Gateway",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "ip",
						Usage: "set LAN ip, example: 192.168.50.2",
					},
					&cli.StringFlag{
						Name:  "mask",
						Value: "255.255.255.0",
						Usage: "set LAN mask, example: 255.255.255.0",
					},
					&cli.StringFlag{
						Name:  "gateway",
						Usage: "set Gateway, example: 192.168.50.1",
					},
					&cli.StringFlag{
						Name:  "dns",
						Usage: "set DNS, example: 192.168.50.1",
					},
					&cli.BoolFlag{
						Name:  "dhcp",
						Value: false,
						Usage: "Enable DHCP or not",
					},
					&cli.BoolFlag{
						Name:  "nat",
						Value: false,
						Usage: "Enable NAT or not",
					},
					&cli.BoolFlag{
						Name:  "json",
						Value: false,
						Usage: "output json",
					},
				},
				Action: func(c *cli.Context) error {
					req := &models.GuideGatewayRouterRequest{
						StaticLanIP: c.String("ip"),
						SubnetMask:  c.String("mask"),
						Gateway:     c.String("gateway"),
						StaticDNSIP: c.String("dns"),
						EnableDhcp:  c.Bool("dhcp"),
						EnableNat:   c.Bool("nat"),
					}
					/* fmt.Println("ip=", req.StaticLanIP,
						"mask=", req.SubnetMask,
						"gateway=", req.Gateway,
						"dns=", req.StaticDNSIP,
						"dhcp=", req.EnableDhcp,
					) */
					_, err := service.SetTransparentGateway(context.TODO(), req)
					if c.Bool("json") {
						result := make(map[string]string)
						if err == nil {
							result["result"] = "OK"
						} else {
							result["result"] = "FAILED"
							result["message"] = err.Error()
						}
						data, _ := json.Marshal(result)
						fmt.Println(string(data))
					} else {
						if err == nil {
							fmt.Println("OK")
						} else {
							fmt.Println("FAILED, err=", err)
						}
					}
					return nil
				},
			},
			{
				Name:  "uciChange",
				Usage: "Run after uci change",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "unix",
						Value: "/var/run/quickstart/local.sock",
					},
				},
				Action: func(ctx *cli.Context) error {
					return networkChange(ctx, "uciChange", nil)
				},
			},
			{
				Name:  "ifaceEvent",
				Usage: "Run when interface changed",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "unix",
						Value: "/var/run/quickstart/local.sock",
					},
				},
				Action: func(ctx *cli.Context) error {
					var args []string
					for i, arg := range os.Args {
						if arg == "ifaceEvent" {
							args = os.Args[i:]
						}
					}
					if len(args) != 3 {
						return errors.New("error input")
					}
					log.Println(args)
					return networkChange(ctx, "ifaceEvent", args[1:])
				},
			},
			{
				Name:  "dhcpValid",
				Usage: "Run to check dhcp",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "unix",
						Value: "/var/run/quickstart/local.sock",
					},
				},
				Action: func(ctx *cli.Context) error {
					err := dhcpValid(ctx)
					if err != nil {
						fmt.Println("dhcp err=", err)
					}
					return nil
				},
			},
			{
				Name:  "dhns",
				Usage: "Run dhcp client in another namespace",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "unix",
						Value: "/var/run/quickstart/local.sock",
					},
				},
				Action: func(c *cli.Context) error {
					cli := dhns.NewDhnsClient(c.String("unix"))
					defer cli.Close()
					select {}
					return nil
				},
			},
		},
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		fmt.Println("run command failed, err=", err)
	}
	return
}

func networkChange(ctx *cli.Context, action string, params []string) error {
	data, err := json.Marshal(&service.DhnsChangeInfo{
		Action: action,
		Params: params,
	})
	if err != nil {
		return err
	}
	// Network change support
	c, err := net.Dial("unix", ctx.String("unix"))
	if err != nil {
		return err
	}
	defer c.Close()
	req, err := http.NewRequest(http.MethodPost, "http://localhost/api/dhns/dhnsChange/", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	err = req.Write(c)
	if err != nil {
		return err
	}
	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("status error code:%d", resp.StatusCode))
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func dhcpValid(ctx *cli.Context) error {
	info := &service.DhcpInfo{
		Ip:      os.Getenv("ip"),
		Gateway: os.Getenv("router"),
		Subnet:  os.Getenv("subnet"),
		Dns:     os.Getenv("dns"),
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	//fmt.Println("info=", string(data))
	c, err := net.Dial("unix", ctx.String("unix"))
	if err != nil {
		return err
	}
	defer c.Close()
	req, err := http.NewRequest(http.MethodPost, "http://localhost/api/dhns/dhcpValid/", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	err = req.Write(c)
	if err != nil {
		return err
	}
	resp, err := http.ReadResponse(bufio.NewReader(c), req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("status error code:%d", resp.StatusCode))
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
