package metadata

import (
	"encoding/json"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

type FileInfo struct {
	ModTimeUnix int64
}

type Store interface {
	Glob(pattern string) ([]string, error)
	Stat(path string) (FileInfo, error)
	ReadFile(path string) ([]byte, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (service *Service) List(root string) ([]*models.AppInstalled, error) {
	globStr := strings.TrimSuffix(root, "/") + "/**.json"
	apps, err := service.store.Glob(globStr)
	if err != nil {
		return nil, err
	}
	modelApps := make([]*models.AppInstalled, 0, len(apps))
	for _, app := range apps {
		fi, err := service.store.Stat(app)
		if err != nil {
			continue
		}
		b, err := service.store.ReadFile(app)
		if err != nil {
			continue
		}
		modelApp := &models.AppInstalled{}
		err = json.Unmarshal(b, modelApp)
		if err != nil {
			continue
		}
		modelApp.Time = fi.ModTimeUnix
		modelApps = append(modelApps, modelApp)
	}
	return modelApps, nil
}
