package thermal

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type Getter interface {
	CPUTemperature() (int, error)
}

type ZoneTemperature struct {
	zoneName string
}

func NewZoneTemperature(zoneName string) ZoneTemperature {
	return ZoneTemperature{zoneName: zoneName}
}

func (temperature ZoneTemperature) CPUTemperature() (int, error) {
	return ReadZoneTemperature(temperature.zoneName)
}

type HwmonTemperature struct {
	hwmon   int
	tempIdx int
}

func NewHwmonTemperature(hwmon int, tempIdx int) HwmonTemperature {
	return HwmonTemperature{hwmon: hwmon, tempIdx: tempIdx}
}

func (temperature HwmonTemperature) CPUTemperature() (int, error) {
	return ReadHwmonTemperature(temperature.hwmon, temperature.tempIdx)
}

type AutocoreTemperature struct{}

func (temperature AutocoreTemperature) CPUTemperature() (int, error) {
	ret, err := utils.BatchOutputCmd(context.Background(), "/sbin/cpuinfo | grep -Eom1 '[ (]\\+?[0-9.]+°C' | head -1 | grep -Eom1 '[0-9.]+'", 0)
	if err == nil {
		msg := strings.Trim(string(ret), "\n")
		temp, err := strconv.ParseFloat(msg, 64)
		if err == nil {
			return int(temp), nil
		}
	}
	return 0, err
}

func BuildTemperatureResult(getter Getter) *models.SystemCPUTemperatureResponseResult {
	temp, err := getter.CPUTemperature()
	if err != nil {
		temp = 0
	}
	return &models.SystemCPUTemperatureResponseResult{Temperature: int64(temp)}
}

func ApplyTemperatureToStatus(status *models.SystemStatusResponseResult, getter Getter) {
	temp, err := getter.CPUTemperature()
	if err != nil {
		temp = 0
	}
	status.CPUTemperature = int64(temp)
}

func ReadZoneTemperature(zoneName string) (int, error) {
	tempPath := fmt.Sprintf("/sys/class/thermal/%v/temp", zoneName)
	ret, err := ioutil.ReadFile(tempPath)
	if err != nil {
		return 0, err
	}
	return parseMilliCelsius(ret), nil
}

func DetectZone() string {
	var tempZone string
	for i := 0; i < 8; i++ {
		filePath := fmt.Sprintf("/sys/class/thermal/thermal_zone%v/type", i)
		ret, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}
		retStr := strings.ToLower(string(ret))
		if strings.Contains(retStr, "x86") ||
			strings.Contains(retStr, "cpu") ||
			strings.Contains(retStr, "soc") {
			tempZone = fmt.Sprintf("thermal_zone%v", i)
			if _, err := ReadZoneTemperature(tempZone); err == nil {
				break
			}
		}
	}
	return tempZone
}

func ReadHwmonTemperature(hwmon int, idx int) (int, error) {
	tempPath := fmt.Sprintf("/sys/class/hwmon/hwmon%d/temp%d_input", hwmon, idx)
	ret, err := ioutil.ReadFile(tempPath)
	if err != nil {
		return 0, err
	}
	return parseMilliCelsius(ret), nil
}

func parseMilliCelsius(raw []byte) int {
	retStr := strings.Trim(string(raw), "\n")
	temp, err := strconv.Atoi(retStr)
	if err != nil {
		temp = 0
	}
	return temp / 1000
}
