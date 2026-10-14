package service

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const hexDigit = "0123456789ABCDEF"

var (
	d                  map[int]map[uint64]string
	manufPrefixLengths []int
	manufStats         ManufRuntimeStats
	initMutex          sync.RWMutex
	initDone           = make(chan struct{})
)

type ManufRuntimeStats struct {
	Entries        int
	SourceBytes    int64
	LoadDuration   time.Duration
	HeapAllocDelta uint64
}

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
	started := time.Now()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	d = make(map[int]map[uint64]string)
	const sourcePath = "/usr/share/quickstart/manuf"
	err := loadManufData(sourcePath)
	if err != nil {
		log.Printf("read manuf file failed, err=%v", err)
	}
	finalizeManufIndex()
	runtime.ReadMemStats(&after)
	manufStats.LoadDuration = time.Since(started)
	if after.HeapAlloc > before.HeapAlloc {
		manufStats.HeapAllocDelta = after.HeapAlloc - before.HeapAlloc
	}
	if info, statErr := os.Stat(sourcePath); statErr == nil {
		manufStats.SourceBytes = info.Size()
	}
	for _, entries := range d {
		manufStats.Entries += len(entries)
	}
	if manufStats.SourceBytes > 0 {
		log.Printf("manuf index loaded entries=%d source_bytes=%d heap_delta=%d duration=%s", manufStats.Entries, manufStats.SourceBytes, manufStats.HeapAllocDelta, manufStats.LoadDuration)
	}
}

func loadManufData(fileName string) error {
	if d == nil {
		d = make(map[int]map[uint64]string)
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
	d[b][maskMACPrefix(value, b)] = comment
}

func finalizeManufIndex() {
	manufPrefixLengths = manufPrefixLengths[:0]
	for prefixBits := range d {
		if prefixBits > 0 && prefixBits <= 48 {
			manufPrefixLengths = append(manufPrefixLengths, prefixBits)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(manufPrefixLengths)))
}

func GomanufDiagnostics() ManufRuntimeStats {
	<-initDone
	return manufStats
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
	for _, prefixBits := range manufPrefixLengths {
		entries := d[prefixBits]
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
