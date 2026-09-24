package service

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const hexDigit = "0123456789ABCDEF"

var (
	d         map[int]interface{}
	initMutex sync.RWMutex
	initDone  = make(chan struct{})
)

func init() {
	go func() {
		initializeManufData()
		close(initDone)
	}()
}

// initializeManufData 初始化制造商数据
func initializeManufData() {
	initMutex.Lock()
	defer initMutex.Unlock()

	d = make(map[int]interface{})
	err := loadManufData("/usr/share/quickstart/manuf")
	if err != nil {
		log.Printf("read manuf file failed, err=%v", err)
	}
}

func loadManufData(fileName string) error {
	if d == nil {
		d = make(map[int]interface{})
	}

	err := readLine(fileName, func(s string) {
		line := strings.Replace(s, "\t\t", "\t", -1)
		l := strings.Split(line, "\t")

		if len(l) > 1 {
			parse(l[0], l[1])
		}
	})
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

func parse(mac, comment string) {
	g := strings.Split(mac, "/")
	m := strings.Split(g[0], ":")
	var b int
	if len(g) != 2 {
		b = len(m) * 8
	} else {
		var err error
		b, err = strconv.Atoi(g[1])
		if err != nil {
			return
		}
	}
	if b <= 0 || b > 48 {
		return
	}
	value, ok := ouiPartsToUint64(m)
	if !ok {
		return
	}
	if _, ok := d[b]; !ok {
		d[b] = make(map[uint64]string)
	}
	d[b].(map[uint64]string)[maskMACPrefix(value, b)] = comment
}

func b2uint64(sList []string) uint64 {
	value, _ := ouiPartsToUint64(sList)
	return value
}

func ouiPartsToUint64(sList []string) (uint64, bool) {
	if len(sList) == 0 || len(sList) > 6 {
		return 0, false
	}
	var t uint64
	for i, b := range sList {
		b = strings.ToUpper(strings.TrimSpace(b))
		if len(b) != 2 {
			return 0, false
		}
		l := strings.IndexByte(hexDigit, b[0])
		r := strings.IndexByte(hexDigit, b[1])
		if l < 0 || r < 0 {
			return 0, false
		}
		t += uint64((l<<4)+r) << uint8((6-i-1)*8)
	}
	return t, true
}

func maskMACPrefix(value uint64, prefixBits int) uint64 {
	return value & ((^uint64(0) << uint(48-prefixBits)) & 0xFFFFFFFFFFFF)
}

// Search 查找MAC地址对应的制造商信息
// 会在首次调用时等待初始化完成
func GomanufSearch(mac string) string {
	// 确保初始化完成
	<-initDone

	hardwareAddr, err := net.ParseMAC(strings.TrimSpace(mac))
	if err != nil || len(hardwareAddr) != 6 {
		return ""
	}
	// Group addresses and locally administered (randomized) addresses do not
	// have a trustworthy globally assigned manufacturer prefix.
	if hardwareAddr[0]&0x03 != 0 {
		return ""
	}

	parts := make([]string, len(hardwareAddr))
	for i, octet := range hardwareAddr {
		parts[i] = string([]byte{hexDigit[octet>>4], hexDigit[octet&0x0F]})
	}
	value, ok := ouiPartsToUint64(parts)
	if !ok {
		return ""
	}

	// Initialization is immutable after initDone closes. Sort prefix lengths
	// explicitly so /36 and /28 assignments always take precedence over /24.
	prefixLengths := make([]int, 0, len(d))
	for prefixBits := range d {
		if prefixBits > 0 && prefixBits <= 48 {
			prefixLengths = append(prefixLengths, prefixBits)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(prefixLengths)))
	for _, prefixBits := range prefixLengths {
		entries, ok := d[prefixBits].(map[uint64]string)
		if !ok {
			continue
		}
		if manufacturer, found := entries[maskMACPrefix(value, prefixBits)]; found {
			return manufacturer
		}
	}
	return ""
}

func readLine(fileName string, handler func(string)) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		line = strings.TrimSpace(line)
		handler(line)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
