package publicaddress

import "github.com/linkease/quick-start/istore-backend/models"

type Reader interface {
	Read() (Snapshot, error)
}

type Classifier interface {
	IsPublic(ipVersion string, address string) bool
}

type Service struct {
	reader     Reader
	classifier Classifier
}

func NewService(reader Reader, classifier Classifier) *Service {
	return &Service{reader: reader, classifier: classifier}
}

func (svc *Service) CheckPublicAddress(ipVersion string) (*models.NetworkCheckPublicNetResponse, error) {
	snapshot, err := svc.reader.Read()
	if err != nil {
		return nil, err
	}

	address, err := selectNetworkPublicAddress(snapshot, ipVersion)
	if err != nil {
		return nil, err
	}

	return buildNetworkPublicAddressResult(address, svc.classifier.IsPublic(ipVersion, address)), nil
}
