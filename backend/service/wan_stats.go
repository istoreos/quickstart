package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	wanStatTimeTick = 5 * time.Second
	slots           = 12
)

type WanStats struct {
	tm        *time.Timer
	isRunning bool
	// Auto stop when no request after 30s
	lastRunningTime time.Time
	lastItem        *NetworkStatisticsItem
	items           []*NetworkStatisticsItem
	fetchStatCh     chan *fetchStatItemsReq
}

type NetworkStatisticsItem struct {
	startTime time.Time
	endTime   time.Time
	txStart   int64
	rxStart   int64
	txAvg     int64
	rxAvg     int64
	device    *string
}

type fetchStatItemsReq struct {
	itemCh chan []*NetworkStatisticsItem
}

func NewWanStats() *WanStats {
	stats := &WanStats{
		lastItem:    &NetworkStatisticsItem{},
		fetchStatCh: make(chan *fetchStatItemsReq, 1),
	}
	go stats.run()
	return stats
}

func (w *WanStats) run() {
	if w.tm != nil {
		return
	}
	w.tm = time.NewTimer(wanStatTimeTick)
	defer w.tm.Stop()
	for {
		if err := w.runOnce(w.tm); err != nil {
			return
		}
	}
}

func (w *WanStats) runOnce(tm *time.Timer) error {
	select {
	case req := <-w.fetchStatCh:
		if w.isRunning {
			w.lastRunningTime = time.Now()
			// Return the olds
			items := make([]*NetworkStatisticsItem, len(w.items))
			copy(items, w.items)
			req.itemCh <- items
			close(req.itemCh)
			return nil
		}
		if !tm.Stop() {
			select {
			case <-tm.C:
			default:
			}
		}
		tm.Reset(wanStatTimeTick)
		w.lastRunningTime = time.Now()
		w.isRunning = true
		err := w.fetchItems()
		if err != nil {
			l.Debugln("fetchItems in fetchStatCh err=", err)
		}

		// Return the news
		items := make([]*NetworkStatisticsItem, len(w.items))
		copy(items, w.items)
		req.itemCh <- items
		close(req.itemCh)

	case <-tm.C:
		now := time.Now()
		if w.lastRunningTime.Add(10 * wanStatTimeTick).Before(now) {
			w.isRunning = false
			tm.Stop()
			return nil
		}
		err := w.fetchItems()
		tm.Reset(wanStatTimeTick)
		if err != nil {
			l.Debugln("fetchItems in timer err=", err)
		}
	}
	return nil
}

func (w *WanStats) fetchItems() error {
	lastItem := w.lastItem
	defaultIf, err := outboundInterface()
	if err != nil {
		return err
	}
	deviceName := defaultIf.deviceName
	if deviceName == "" {
		deviceName = defaultIf.l3Device
	}
	if deviceName == "" {
		return errors.New("no device name")
	}
	rx, tx, err := w.watch(deviceName)
	if err != nil {
		return err
	}

	curr := time.Now()
	if lastItem.device == nil || *lastItem.device != deviceName || rx < lastItem.rxStart || tx < lastItem.txStart {
		lastItem.device = &deviceName
		lastItem.rxStart = rx
		lastItem.txStart = tx
		lastItem.startTime = curr
		w.lastItem = lastItem
		// Reset lastItem
		return nil
	}

	lastItem.endTime = curr
	duration := lastItem.endTime.Sub(lastItem.startTime).Milliseconds() + 1
	lastItem.rxAvg = 1000 * (rx - lastItem.rxStart) / int64(duration)
	lastItem.txAvg = 1000 * (tx - lastItem.txStart) / int64(duration)
	if len(w.items) >= slots {
		// w.items = 12
		// 12+1-12 = 1, w.items = w.items[1:]
		//w.items = w.items[len(w.items)+1-slots:]
		for i := 0; i < slots-1; i++ {
			w.items[i] = w.items[i+1]
		}
		w.items = w.items[:slots-1]
	}
	w.items = append(w.items, lastItem)

	lastItem = &NetworkStatisticsItem{
		startTime: curr,
		device:    &deviceName,
		rxStart:   rx,
		txStart:   tx,
	}
	w.lastItem = lastItem
	return nil

}

func (w *WanStats) watch(deviceName string) (int64, int64, error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	status, err := UbusCall(ctx, fmt.Sprintf("network.device status {\"name\":\"%s\"}", deviceName))
	if err != nil {
		return 0, 0, err
	}

	rx_bytes, err := status.Get("statistics").Get("rx_bytes").Int64()
	if err != nil {
		return 0, 0, err
	}
	tx_bytes, err := status.Get("statistics").Get("tx_bytes").Int64()
	if err != nil {
		return rx_bytes, 0, err
	}

	// l.Debugln("device: ", device, "tx_bytes: ", tx_bytes, ", rx_bytes: ", rx_bytes)

	return rx_bytes, tx_bytes, nil
}

func (w *WanStats) GetItems() []*NetworkStatisticsItem {
	fetchItemReq := &fetchStatItemsReq{
		itemCh: make(chan []*NetworkStatisticsItem, 1),
	}
	w.fetchStatCh <- fetchItemReq
	items := <-fetchItemReq.itemCh
	return items
}
