package utils

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

const (
	ChineseTimeLayout = "2006-01-02 15:04:05"
	PythonLayout      = "2006-01-02T15:04:05.999999Z"
)

var weekmaps = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

func TimeToTick(t time.Time) int64 {
	return int64(t.Unix()*1e3) + int64(t.Nanosecond()/1e6)
}

func TickToTime(tick int64) time.Time {
	return time.Unix(tick/1e3, (tick%1e3)*1e6)
}

func TimeToString(t time.Time) string {
	tick := int64(t.Unix()*1e3) + int64(t.Nanosecond()/1e6)
	return strconv.FormatInt(tick, 10)
}

func TimeToWeekString(t time.Time) string {
	now := time.Now()
	if now.Year() == t.Year() {
		return fmt.Sprintf("%02d月%02d日 %s", t.Month(), t.Day(), weekmaps[t.Weekday()])
	} else {
		return fmt.Sprintf("%d年%02d月%02d日 %s", t.Year(), t.Month(), t.Day(), weekmaps[t.Weekday()])
	}
}

func TimeToDayString(t time.Time) string {
	now := time.Now()
	yes := now.Add(0 - time.Hour*24)
	if now.Year() == t.Year() {
		d := now.Sub(t)
		if d.Seconds() < 60 {
			return strconv.Itoa(int(d.Seconds())) + "秒前"
		} else if d.Seconds() < 60*60 {
			return strconv.Itoa(int(d.Seconds()/60)) + "分前"
		} else if d.Hours() < 24 && now.Day() == t.Day() {
			return fmt.Sprintf("今天%02d:%02d", t.Hour(), t.Minute())
		} else if d.Hours() < 48 && now.Day() == yes.Day() {
			return fmt.Sprintf("昨天%02d:%02d", t.Hour(), t.Minute())
		} else {
			return fmt.Sprintf("%02d月%02d日", t.Month(), t.Day())
		}
	} else {
		return fmt.Sprintf("%d年%02d月%02d日", t.Year(), t.Month(), t.Day())
	}
}

func StringToTime(s string) (time.Time, error) {
	tick, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(tick/1e3, (tick%1e3)*1e6), nil
}

func NormalizeTime(t time.Time) time.Time {
	return time.Unix(t.Unix(), int64(t.Nanosecond()/1e6))
}

func plural(count int, singular string) (result string) {
	result = strconv.Itoa(count) + singular + " "
	return
}

func SecondsToHuman(input int64) (result string) {
	years := math.Floor(float64(input) / 60 / 60 / 24 / 7 / 30 / 12)
	seconds := input % (60 * 60 * 24 * 7 * 30 * 12)
	months := math.Floor(float64(seconds) / 60 / 60 / 24 / 7 / 30)
	seconds = input % (60 * 60 * 24 * 7 * 30)
	weeks := math.Floor(float64(seconds) / 60 / 60 / 24 / 7)
	seconds = input % (60 * 60 * 24 * 7)
	days := math.Floor(float64(seconds) / 60 / 60 / 24)
	seconds = input % (60 * 60 * 24)
	hours := math.Floor(float64(seconds) / 60 / 60)
	seconds = input % (60 * 60)
	minutes := math.Floor(float64(seconds) / 60)
	seconds = input % 60

	if years > 0 {
		result = plural(int(years), "y") + plural(int(months), "m") + plural(int(weeks), "w") + plural(int(days), "d") + plural(int(hours), "h") + plural(int(minutes), "m") + plural(int(seconds), "s")
	} else if months > 0 {
		result = plural(int(months), "m") + plural(int(weeks), "w") + plural(int(days), "d") + plural(int(hours), "h") + plural(int(minutes), "m") + plural(int(seconds), "s")
	} else if weeks > 0 {
		result = plural(int(weeks), "w") + plural(int(days), "d") + plural(int(hours), "h") + plural(int(minutes), "m") + plural(int(seconds), "s")
	} else if days > 0 {
		result = plural(int(days), "d") + plural(int(hours), "h") + plural(int(minutes), "m") + plural(int(seconds), "s")
	} else if hours > 0 {
		result = plural(int(hours), "h") + plural(int(minutes), "m") + plural(int(seconds), "s")
	} else if minutes > 0 {
		result = plural(int(minutes), "m") + plural(int(seconds), "s")
	} else {
		result = plural(int(seconds), "s")
	}

	return
}
