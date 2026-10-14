package service

import (
	"encoding/json"
	"errors"
	"os"
	"sort"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	defaultTrafficInsightsPath = "/etc/quickstart/traffic-insights.json"
	trafficInsightsBudget      = 2 * 1024 * 1024
	trafficInsightsDeviceLimit = 2048
)

type trafficCounterBaseline struct {
	UploadBytes   int64 `json:"uploadBytes"`
	DownloadBytes int64 `json:"downloadBytes"`
}

type trafficQuotaRuntime struct {
	PeriodStart   string `json:"periodStart,omitempty"`
	Blocked       bool   `json:"blocked,omitempty"`
	RestoreAccess bool   `json:"restoreAccess,omitempty"`
	Notified      bool   `json:"notified,omitempty"`
}

type trafficInsightsDocument struct {
	SchemaVersion int                                       `json:"schemaVersion"`
	Hourly        map[string][]*models.TrafficInsightBucket `json:"hourly"`
	Daily         map[string][]*models.TrafficInsightBucket `json:"daily"`
	Monthly       map[string][]*models.TrafficInsightBucket `json:"monthly"`
	Baselines     map[string]trafficCounterBaseline         `json:"baselines"`
	BaselineSeen  map[string]string                         `json:"baselineSeen"`
	Quotas        map[string]*models.TrafficQuota           `json:"quotas"`
	QuotaRuntime  map[string]trafficQuotaRuntime            `json:"quotaRuntime"`
}

type trafficInsightsFileStore struct {
	path    string
	persist func(string, []byte) error
}

func newTrafficInsightsFileStore(path string) *trafficInsightsFileStore {
	return &trafficInsightsFileStore{path: path, persist: persistClassificationOverrides}
}

func (store *trafficInsightsFileStore) Load() (trafficInsightsDocument, int64, error) {
	raw, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyTrafficInsightsDocument(), 0, nil
	}
	if err != nil {
		return trafficInsightsDocument{}, 0, err
	}
	var document trafficInsightsDocument
	if err := json.Unmarshal(raw, &document); err != nil || document.SchemaVersion != 1 {
		return trafficInsightsDocument{}, 0, errors.New("invalid traffic insights file")
	}
	normalizeTrafficInsightsDocument(&document)
	return document, int64(len(raw)), nil
}

func (store *trafficInsightsFileStore) Save(document *trafficInsightsDocument) (int64, error) {
	pruneTrafficDocumentToBudget(document, trafficInsightsBudget)
	raw, err := json.Marshal(document)
	if err != nil {
		return 0, err
	}
	if len(raw) > trafficInsightsBudget {
		return 0, errors.New("traffic insights storage budget exceeded")
	}
	if err := store.persist(store.path, raw); err != nil {
		return 0, err
	}
	return int64(len(raw)), nil
}

func emptyTrafficInsightsDocument() trafficInsightsDocument {
	return trafficInsightsDocument{
		SchemaVersion: 1,
		Hourly:        map[string][]*models.TrafficInsightBucket{}, Daily: map[string][]*models.TrafficInsightBucket{},
		Monthly: map[string][]*models.TrafficInsightBucket{}, Baselines: map[string]trafficCounterBaseline{}, BaselineSeen: map[string]string{},
		Quotas: map[string]*models.TrafficQuota{}, QuotaRuntime: map[string]trafficQuotaRuntime{},
	}
}

func normalizeTrafficInsightsDocument(document *trafficInsightsDocument) {
	if document.Hourly == nil {
		document.Hourly = map[string][]*models.TrafficInsightBucket{}
	}
	if document.Daily == nil {
		document.Daily = map[string][]*models.TrafficInsightBucket{}
	}
	if document.Monthly == nil {
		document.Monthly = map[string][]*models.TrafficInsightBucket{}
	}
	if document.Baselines == nil {
		document.Baselines = map[string]trafficCounterBaseline{}
	}
	if document.BaselineSeen == nil {
		document.BaselineSeen = map[string]string{}
	}
	if document.Quotas == nil {
		document.Quotas = map[string]*models.TrafficQuota{}
	}
	if document.QuotaRuntime == nil {
		document.QuotaRuntime = map[string]trafficQuotaRuntime{}
	}
}

func pruneTrafficDocumentToBudget(document *trafficInsightsDocument, budget int) {
	for {
		raw, _ := json.Marshal(document)
		if len(raw) <= budget {
			return
		}
		if removeOldestTrafficBucket(document.Hourly) {
			continue
		}
		if removeOldestTrafficBucket(document.Daily) {
			continue
		}
		if removeOldestTrafficBucket(document.Monthly) {
			continue
		}
		return
	}
}

func removeOldestTrafficBucket(series map[string][]*models.TrafficInsightBucket) bool {
	oldestDevice, oldestStart := "", ""
	for deviceID, buckets := range series {
		if len(buckets) == 0 {
			continue
		}
		if oldestStart == "" || buckets[0].Start < oldestStart {
			oldestDevice, oldestStart = deviceID, buckets[0].Start
		}
	}
	if oldestDevice == "" {
		return false
	}
	series[oldestDevice] = series[oldestDevice][1:]
	if len(series[oldestDevice]) == 0 {
		delete(series, oldestDevice)
	}
	return true
}

func sortedTrafficBuckets(values []*models.TrafficInsightBucket) []*models.TrafficInsightBucket {
	result := append([]*models.TrafficInsightBucket(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i].Start < result[j].Start })
	return result
}
