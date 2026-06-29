package thermal

import (
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeGetter struct {
	temp int
	err  error
}

func (getter fakeGetter) CPUTemperature() (int, error) {
	return getter.temp, getter.err
}

func fakeStatusResult() *models.SystemStatusResponseResult {
	return &models.SystemStatusResponseResult{
		CPUUsage: 12,
		MemTotal: "512MB",
	}
}

func TestBuildTemperatureResultReturnsTemperature(t *testing.T) {
	t.Parallel()

	result := BuildTemperatureResult(fakeGetter{temp: 47})

	if result.Temperature != 47 {
		t.Fatalf("Temperature = %d, want 47", result.Temperature)
	}
}

func TestBuildTemperatureResultFallsBackToZeroOnError(t *testing.T) {
	t.Parallel()

	result := BuildTemperatureResult(fakeGetter{err: errors.New("read failed")})

	if result.Temperature != 0 {
		t.Fatalf("Temperature = %d, want 0", result.Temperature)
	}
}

func TestApplyTemperatureToStatus(t *testing.T) {
	t.Parallel()

	status := fakeStatusResult()

	ApplyTemperatureToStatus(status, fakeGetter{temp: 52})

	if status.CPUTemperature != 52 {
		t.Fatalf("CPUTemperature = %d, want 52", status.CPUTemperature)
	}
}

func TestApplyTemperatureToStatusFallsBackToZeroOnError(t *testing.T) {
	t.Parallel()

	status := fakeStatusResult()

	ApplyTemperatureToStatus(status, fakeGetter{err: errors.New("read failed")})

	if status.CPUTemperature != 0 {
		t.Fatalf("CPUTemperature = %d, want 0", status.CPUTemperature)
	}
}

func TestParseMilliCelsius(t *testing.T) {
	t.Parallel()

	temp := parseMilliCelsius([]byte("47000\n"))

	if temp != 47 {
		t.Fatalf("temp = %d, want 47", temp)
	}
}

func TestParseMilliCelsiusReturnsZeroForInvalidInput(t *testing.T) {
	t.Parallel()

	temp := parseMilliCelsius([]byte("not-a-number\n"))

	if temp != 0 {
		t.Fatalf("temp = %d, want 0", temp)
	}
}
