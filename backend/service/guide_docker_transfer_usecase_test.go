package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
	dockertransfer "github.com/linkease/quick-start/istore-backend/modules/guidestorage/dockertransfer"
)

type fakeGuideDockerTransferFacade struct {
	partitionListResp  *models.GuideDockerPartitionListResponse
	partitionListErr   error
	partitionListCalls int

	transferResp   *models.GuideDockerTransferResponse
	transferErr    error
	transferInputs []GuideDockerTransferInput
}

func (facade *fakeGuideDockerTransferFacade) GetPartitionList(ctx context.Context) (*models.GuideDockerPartitionListResponse, error) {
	facade.partitionListCalls++
	return facade.partitionListResp, facade.partitionListErr
}

func (facade *fakeGuideDockerTransferFacade) Transfer(ctx context.Context, input GuideDockerTransferInput) (*models.GuideDockerTransferResponse, error) {
	facade.transferInputs = append(facade.transferInputs, input)
	return facade.transferResp, facade.transferErr
}

type fakeGuideDockerTransferReader struct {
	root       *GuideDockerRootSnapshot
	rootErr    error
	candidates []*GuideDockerPartitionCandidate
	candErr    error
}

func (reader *fakeGuideDockerTransferReader) ReadDockerRootPath(ctx context.Context) (*GuideDockerRootSnapshot, error) {
	return reader.root, reader.rootErr
}

func (reader *fakeGuideDockerTransferReader) ReadPartitionCandidates(ctx context.Context) ([]*GuideDockerPartitionCandidate, error) {
	return reader.candidates, reader.candErr
}

type fakeGuideDockerTransferWriter struct {
	validateTarget string
	validateOrigin string
	validateErr    error

	transferTarget    string
	transferForce     bool
	transferOverwrite bool
	transferOrigin    string
	transferResult    *models.GuideDockerTransferResponseResult
	transferErr       error

	updatePath string
	updateErr  error
}

func (writer *fakeGuideDockerTransferWriter) ValidateTargetPath(ctx context.Context, targetPath string, originPath string) error {
	writer.validateTarget = targetPath
	writer.validateOrigin = originPath
	return writer.validateErr
}

func (writer *fakeGuideDockerTransferWriter) TransferPath(ctx context.Context, targetPath string, force bool, overwriteDir bool, originPath string) (*models.GuideDockerTransferResponseResult, error) {
	writer.transferTarget = targetPath
	writer.transferForce = force
	writer.transferOverwrite = overwriteDir
	writer.transferOrigin = originPath
	return writer.transferResult, writer.transferErr
}

func (writer *fakeGuideDockerTransferWriter) UpdateDockerRootPath(ctx context.Context, path string) error {
	writer.updatePath = path
	return writer.updateErr
}

func TestGuideDockerTransferServiceBuildsPartitionListResponse(t *testing.T) {
	service := GuideDockerTransferService{
		reader: &fakeGuideDockerTransferReader{
			candidates: []*GuideDockerPartitionCandidate{
				{Path: "/mnt/a/docker"},
				{Path: "/mnt/b/docker"},
			},
		},
	}

	resp, err := service.GetPartitionList(context.Background())
	if err != nil {
		t.Fatalf("unexpected partition-list error: %v", err)
	}
	if resp == nil || resp.Result == nil {
		t.Fatalf("expected partition-list payload, got %#v", resp)
	}
	if !reflect.DeepEqual(resp.Result.PartitionList, []string{"/mnt/a/docker", "/mnt/b/docker"}) {
		t.Fatalf("unexpected partition-list result: %#v", resp.Result.PartitionList)
	}
}

func TestGuideDockerTransferServiceTransferPreservesLegacyBranching(t *testing.T) {
	writer := &fakeGuideDockerTransferWriter{}
	service := GuideDockerTransferService{
		reader: &fakeGuideDockerTransferReader{
			root: &GuideDockerRootSnapshot{Path: "/mnt/origin/docker"},
		},
		writer: writer,
	}

	writer.transferResult = &models.GuideDockerTransferResponseResult{
		Path:             "/mnt/new/docker",
		EmptyPathWarning: true,
	}
	writer.transferErr = dockertransfer.ErrEmptyTargetDirectory
	resp, err := service.Transfer(context.Background(), GuideDockerTransferInput{
		Path:         "/mnt/new/docker",
		Force:        false,
		OverwriteDir: false,
	})
	if err != nil {
		t.Fatalf("expected warning branch to return response without error, got %v", err)
	}
	if resp == nil || resp.Result == nil || !resp.Result.EmptyPathWarning || resp.Result.Path != "/mnt/new/docker" {
		t.Fatalf("unexpected warning response: %#v", resp)
	}
	if writer.updatePath != "" {
		t.Fatalf("docker root path should not update on warning branch: %q", writer.updatePath)
	}

	writer.transferResult = nil
	writer.transferErr = nil
	resp, err = service.Transfer(context.Background(), GuideDockerTransferInput{
		Path:         "/mnt/new/docker",
		Force:        true,
		OverwriteDir: true,
	})
	if err != nil {
		t.Fatalf("unexpected transfer success error: %v", err)
	}
	if resp == nil || resp.Result != nil {
		t.Fatalf("expected empty success response, got %#v", resp)
	}
	if writer.validateTarget != "/mnt/new/docker" || writer.validateOrigin != "/mnt/origin/docker" {
		t.Fatalf("unexpected validate delegation: target=%q origin=%q", writer.validateTarget, writer.validateOrigin)
	}
	if writer.transferTarget != "/mnt/new/docker" || !writer.transferForce || !writer.transferOverwrite || writer.transferOrigin != "/mnt/origin/docker" {
		t.Fatalf("unexpected transfer delegation: %#v", writer)
	}
	if writer.updatePath != "/mnt/new/docker" {
		t.Fatalf("expected docker root update, got %q", writer.updatePath)
	}
}

func TestGuideDockerTransferServicePropagatesReaderAndWriterErrors(t *testing.T) {
	rootErr := errors.New("root failed")
	service := GuideDockerTransferService{
		reader: &fakeGuideDockerTransferReader{
			rootErr: rootErr,
		},
		writer: &fakeGuideDockerTransferWriter{},
	}
	if _, err := service.Transfer(context.Background(), GuideDockerTransferInput{Path: "/mnt/new/docker"}); !errors.Is(err, rootErr) {
		t.Fatalf("expected root read error, got %v", err)
	}

	candidateErr := errors.New("candidates failed")
	service.reader = &fakeGuideDockerTransferReader{candErr: candidateErr}
	if _, err := service.GetPartitionList(context.Background()); !errors.Is(err, candidateErr) {
		t.Fatalf("expected candidate read error, got %v", err)
	}

	validateErr := errors.New("validate failed")
	service.reader = &fakeGuideDockerTransferReader{root: &GuideDockerRootSnapshot{Path: "/mnt/origin/docker"}}
	service.writer = &fakeGuideDockerTransferWriter{validateErr: validateErr}
	if _, err := service.Transfer(context.Background(), GuideDockerTransferInput{Path: "/mnt/new/docker"}); !errors.Is(err, validateErr) {
		t.Fatalf("expected validate error, got %v", err)
	}

	transferErr := errors.New("copy failed")
	service.writer = &fakeGuideDockerTransferWriter{transferErr: transferErr}
	if _, err := service.Transfer(context.Background(), GuideDockerTransferInput{Path: "/mnt/new/docker"}); !errors.Is(err, transferErr) {
		t.Fatalf("expected transfer error, got %v", err)
	}

	updateErr := errors.New("update failed")
	service.writer = &fakeGuideDockerTransferWriter{updateErr: updateErr}
	if _, err := service.Transfer(context.Background(), GuideDockerTransferInput{Path: "/mnt/new/docker"}); !errors.Is(err, updateErr) {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestServiceBackendGetGuideDockerPartitionListCompatibility(t *testing.T) {
	orig := newGuideDockerTransferFacade
	defer func() { newGuideDockerTransferFacade = orig }()

	facade := &fakeGuideDockerTransferFacade{
		partitionListResp: &models.GuideDockerPartitionListResponse{
			Result: &models.GuideDockerPartitionListResponseResult{
				PartitionList: []string{"/mnt/a/docker", "/mnt/b/docker"},
			},
		},
	}
	newGuideDockerTransferFacade = func() guideDockerTransferFacade { return facade }

	resp, err := (&ServiceBackend{}).GetGuideDockerPartList(context.Background())
	if err != nil {
		t.Fatalf("unexpected GetGuideDockerPartList error: %v", err)
	}
	if facade.partitionListCalls != 1 {
		t.Fatalf("expected one partition-list call, got %d", facade.partitionListCalls)
	}
	if !reflect.DeepEqual(resp, facade.partitionListResp) {
		t.Fatalf("expected passthrough response, got %#v", resp)
	}
}

func TestServiceBackendGetGuideDockerPartitionListCompatibilityPropagatesErrors(t *testing.T) {
	orig := newGuideDockerTransferFacade
	defer func() { newGuideDockerTransferFacade = orig }()

	serviceErr := errors.New("partition list failed")
	newGuideDockerTransferFacade = func() guideDockerTransferFacade {
		return &fakeGuideDockerTransferFacade{partitionListErr: serviceErr}
	}

	if _, err := (&ServiceBackend{}).GetGuideDockerPartList(context.Background()); !errors.Is(err, serviceErr) {
		t.Fatalf("expected GetGuideDockerPartList error, got %v", err)
	}
}

func TestServiceBackendPostGuideDockerTransferCompatibility(t *testing.T) {
	orig := newGuideDockerTransferFacade
	defer func() { newGuideDockerTransferFacade = orig }()

	facade := &fakeGuideDockerTransferFacade{
		transferResp: &models.GuideDockerTransferResponse{
			Result: &models.GuideDockerTransferResponseResult{
				Path:             "/mnt/new/docker",
				EmptyPathWarning: true,
			},
		},
	}
	newGuideDockerTransferFacade = func() guideDockerTransferFacade { return facade }

	req := httptest.NewRequest("POST", "/guide/docker-transfer", strings.NewReader(`{"path":"/mnt/new/docker","force":true,"overwriteDir":false}`))
	resp, err := (&ServiceBackend{}).PostGuideDockerTransfer(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected PostGuideDockerTransfer error: %v", err)
	}
	if len(facade.transferInputs) != 1 {
		t.Fatalf("expected one transfer call, got %d", len(facade.transferInputs))
	}
	if facade.transferInputs[0].Path != "/mnt/new/docker" || !facade.transferInputs[0].Force || facade.transferInputs[0].OverwriteDir {
		t.Fatalf("unexpected transfer delegation input: %#v", facade.transferInputs[0])
	}
	if !reflect.DeepEqual(resp, facade.transferResp) {
		t.Fatalf("expected passthrough response, got %#v", resp)
	}
}

func TestServiceBackendPostGuideDockerTransferCompatibilityPropagatesErrors(t *testing.T) {
	orig := newGuideDockerTransferFacade
	defer func() { newGuideDockerTransferFacade = orig }()

	serviceErr := errors.New("docker transfer failed")
	newGuideDockerTransferFacade = func() guideDockerTransferFacade {
		return &fakeGuideDockerTransferFacade{transferErr: serviceErr}
	}

	req := httptest.NewRequest("POST", "/guide/docker-transfer", strings.NewReader(`{"path":"/mnt/new/docker","force":false,"overwriteDir":false}`))
	if _, err := (&ServiceBackend{}).PostGuideDockerTransfer(context.Background(), req); !errors.Is(err, serviceErr) {
		t.Fatalf("expected PostGuideDockerTransfer error, got %v", err)
	}
}

func TestDockerTransferToolCompatibility(t *testing.T) {
	orig := newGuideDockerTransferFacade
	defer func() { newGuideDockerTransferFacade = orig }()

	facade := &fakeGuideDockerTransferFacade{
		transferResp: &models.GuideDockerTransferResponse{},
	}
	newGuideDockerTransferFacade = func() guideDockerTransferFacade { return facade }

	if err := DockerTransferTool("/mnt/new/docker"); err != nil {
		t.Fatalf("unexpected DockerTransferTool error: %v", err)
	}
	if len(facade.transferInputs) != 1 {
		t.Fatalf("expected one transfer call, got %d", len(facade.transferInputs))
	}
	if facade.transferInputs[0].Path != "/mnt/new/docker" || !facade.transferInputs[0].Force || facade.transferInputs[0].OverwriteDir {
		t.Fatalf("unexpected DockerTransferTool delegation input: %#v", facade.transferInputs[0])
	}
}

func TestDockerTransferToolCompatibilityPreservesWarningAndErrorSemantics(t *testing.T) {
	orig := newGuideDockerTransferFacade
	defer func() { newGuideDockerTransferFacade = orig }()

	newGuideDockerTransferFacade = func() guideDockerTransferFacade {
		return &fakeGuideDockerTransferFacade{
			transferResp: &models.GuideDockerTransferResponse{
				Result: &models.GuideDockerTransferResponseResult{
					Path:             "/mnt/new/docker",
					EmptyPathWarning: true,
				},
			},
		}
	}
	if err := DockerTransferTool("/mnt/new/docker"); err == nil || err.Error() != "目标路径不为空" {
		t.Fatalf("expected DockerTransferTool warning error, got %v", err)
	}

	serviceErr := errors.New("docker transfer failed")
	newGuideDockerTransferFacade = func() guideDockerTransferFacade {
		return &fakeGuideDockerTransferFacade{transferErr: serviceErr}
	}
	if err := DockerTransferTool("/mnt/new/docker"); !errors.Is(err, serviceErr) {
		t.Fatalf("expected DockerTransferTool error, got %v", err)
	}
}
