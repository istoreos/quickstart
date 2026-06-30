package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

const istorePrefix = "/cgi-bin/luci/istore"

type routerClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

type envelope struct {
	Success *int   `json:"success"`
	Error   string `json:"error"`
	Scope   string `json:"scope"`
	Detail  string `json:"detail"`
	Result  any    `json:"result"`
}

func normalizeBaseURL(host, scheme string) (*url.URL, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, errors.New("host is required")
	}
	if !strings.Contains(host, "://") {
		if scheme == "" {
			scheme = "http"
		}
		host = scheme + "://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid host %q", host)
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u, nil
}

func quickstartURL(base *url.URL, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasPrefix(path, istorePrefix+"/") && path != istorePrefix {
		path = istorePrefix + path
	}
	u := *base
	u.Path = path
	return u.String()
}

func newRouterClient(host, scheme string, insecure bool) (*routerClient, error) {
	base, err := normalizeBaseURL(host, scheme)
	if err != nil {
		return nil, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &routerClient{
		baseURL: base,
		httpClient: &http.Client{
			Jar:       jar,
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}, nil
}

func (c *routerClient) login(ctx context.Context, username, password string) error {
	if strings.TrimSpace(username) == "" {
		return errors.New("username is required")
	}
	form := url.Values{}
	form.Set("luci_username", username)
	form.Set("luci_password", password)

	loginURL := *c.baseURL
	loginURL.Path = "/cgi-bin/luci/"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("login failed: HTTP %s", resp.Status)
	}
	if !c.hasAuthCookie() {
		return errors.New("login failed: no LuCI sysauth cookie returned")
	}
	return nil
}

func (c *routerClient) hasAuthCookie() bool {
	for _, cookie := range c.httpClient.Jar.Cookies(c.baseURL) {
		switch cookie.Name {
		case "sysauth", "sysauth_http", "sysauth_https":
			if cookie.Value != "" {
				return true
			}
		}
	}
	return false
}

func (c *routerClient) doQuickstart(ctx context.Context, method, path, body string) ([]byte, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return nil, errors.New("method is required")
	}
	var reader io.Reader
	if body != "" {
		if !json.Valid([]byte(body)) {
			return nil, errors.New("request body must be valid JSON")
		}
		reader = bytes.NewBufferString(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, quickstartURL(c.baseURL, path), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != "" {
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return data, fmt.Errorf("quickstart request failed: HTTP %s", resp.Status)
	}
	if err := checkEnvelope(data); err != nil {
		return data, err
	}
	return data, nil
}

func checkEnvelope(data []byte) error {
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("response is not JSON: %w", err)
	}
	if env.Success != nil && *env.Success != 0 {
		parts := []string{fmt.Sprintf("quickstart error success=%d", *env.Success)}
		if env.Scope != "" {
			parts = append(parts, "scope="+env.Scope)
		}
		if env.Error != "" {
			parts = append(parts, "error="+env.Error)
		}
		if env.Detail != "" {
			parts = append(parts, "detail="+env.Detail)
		}
		return errors.New(strings.Join(parts, " "))
	}
	return nil
}

func resolvePassword(password, passwordEnv string, getenv func(string) string) (string, error) {
	if password != "" && passwordEnv != "" {
		return "", errors.New("use only one of --password or --password-env")
	}
	if passwordEnv != "" {
		value := getenv(passwordEnv)
		if value == "" {
			return "", fmt.Errorf("environment variable %s is empty or unset", passwordEnv)
		}
		return value, nil
	}
	return password, nil
}

func requireConfirmation(method, path string, confirmWrite, confirmDanger bool, getenv func(string) string) error {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method != http.MethodPost {
		return nil
	}
	if !confirmWrite && getenv("CONFIRM_QSCTL_WRITE") != "YES" {
		return errors.New("POST requires --confirm-write or CONFIRM_QSCTL_WRITE=YES after user confirmation")
	}
	if isDangerousPath(path) && !confirmDanger && getenv("CONFIRM_QSCTL_DANGER") != "YES" {
		return errors.New("dangerous endpoint requires --confirm-danger or CONFIRM_QSCTL_DANGER=YES after explicit user confirmation")
	}
	return nil
}

func isDangerousPath(path string) bool {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if strings.HasPrefix(path, istorePrefix+"/") {
		path = strings.TrimPrefix(path, istorePrefix)
	}
	path = strings.TrimSuffix(path, "/")
	dangerous := map[string]struct{}{
		"/system/reboot":              {},
		"/system/poweroff":            {},
		"/system/setPassword":         {},
		"/network/interface/config":   {},
		"/nas/disk/init":              {},
		"/nas/disk/initrest":          {},
		"/nas/disk/partition/format":  {},
		"/nas/disk/partition/mount":   {},
		"/nas/sandbox/commit":         {},
		"/nas/sandbox/reset":          {},
		"/raid/create":                {},
		"/raid/delete":                {},
		"/raid/add":                   {},
		"/raid/remove":                {},
		"/raid/recover":               {},
		"/app/install":                {},
		"/wireless/enable-iface":      {},
		"/wireless/set-device-power":  {},
		"/wireless/edit-iface":        {},
		"/wireless/setup":             {},
		"/lanctrl/enableSpeedLimit":   {},
		"/lanctrl/enableFloatGateway": {},
		"/lanctrl/staticDeviceConfig": {},
		"/lanctrl/speedLimitConfig":   {},
		"/lanctrl/dhcpTagsConfig":     {},
		"/lanctrl/dhcpGatewayConfig":  {},
		"/share/user/delete":          {},
		"/share/service/delete":       {},
		"/share/protocol/webdav":      {},
		"/share/protocol/samba":       {},
		"/share/protocol/globals":     {},
		"/guide/gateway-router":       {},
		"/guide/lan":                  {},
		"/guide/client-mode":          {},
		"/guide/pppoe":                {},
		"/guide/dns-config":           {},
		"/guide/docker/transfer":      {},
		"/guide/aria2/init":           {},
		"/guide/qbittorrent/init":     {},
		"/guide/transmission/init":    {},
	}
	_, ok := dangerous[path]
	return ok
}

func main() {
	var host string
	var scheme string
	var username string
	var password string
	var passwordEnv string
	var insecure bool
	var noLogin bool
	var confirmWrite bool
	var confirmDanger bool

	flag.StringVar(&host, "host", "", "Router host or base URL, for example 192.168.30.244 or https://router.lan")
	flag.StringVar(&scheme, "scheme", "http", "Default scheme when --host has no scheme")
	flag.StringVar(&username, "user", "", "LuCI username")
	flag.StringVar(&password, "password", "", "LuCI password; prefer --password-env to avoid exposing secrets in process argv")
	flag.StringVar(&passwordEnv, "password-env", "", "Environment variable containing the LuCI password")
	flag.BoolVar(&insecure, "insecure", false, "Skip TLS certificate verification for HTTPS")
	flag.BoolVar(&noLogin, "no-login", false, "Skip LuCI login; useful only when testing an unprotected local endpoint")
	flag.BoolVar(&confirmWrite, "confirm-write", false, "Confirm the user approved this POST request")
	flag.BoolVar(&confirmDanger, "confirm-danger", false, "Confirm the user explicitly approved a dangerous endpoint")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s --host HOST --user USER --password-env ENV get PATH\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "       %s --host HOST --user USER --password-env ENV --confirm-write post PATH JSON\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		flag.Usage()
		os.Exit(2)
	}
	command := strings.ToLower(args[0])
	path := args[1]
	body := ""
	method := ""
	switch command {
	case "get":
		method = http.MethodGet
		if len(args) != 2 {
			exitErr("get expects exactly PATH")
		}
	case "post":
		method = http.MethodPost
		if len(args) != 3 {
			exitErr("post expects PATH and JSON")
		}
		body = args[2]
	default:
		exitErr("command must be get or post")
	}
	password, err := resolvePassword(password, passwordEnv, os.Getenv)
	if err != nil {
		exitErr(err.Error())
	}
	if err := requireConfirmation(method, path, confirmWrite, confirmDanger, os.Getenv); err != nil {
		exitErr(err.Error())
	}

	client, err := newRouterClient(host, scheme, insecure)
	if err != nil {
		exitErr(err.Error())
	}
	ctx := context.Background()
	if !noLogin {
		if err := client.login(ctx, username, password); err != nil {
			exitErr(err.Error())
		}
	}
	data, err := client.doQuickstart(ctx, method, path, body)
	if len(data) > 0 {
		_, _ = os.Stdout.Write(data)
		if data[len(data)-1] != '\n' {
			_, _ = os.Stdout.WriteString("\n")
		}
	}
	if err != nil {
		exitErr(err.Error())
	}
}

func exitErr(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
