package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSmartCommandFacade struct {
	testReq      *models.SmartTestRequest
	testResult   *models.SmartTestResultRequest
	attributeReq *models.SmartAttributeResultRequest
	extendReq    *models.SmartExtendResultRequest
}

func (facade *fakeSmartCommandFacade) StartTest(ctx context.Context, req models.SmartTestRequest) (*models.SmartTestResponse, error) {
	facade.testReq = &req
	return &models.SmartTestResponse{Result: &models.SmartTestResponseResult{Result: "started"}}, nil
}

func (facade *fakeSmartCommandFacade) TestResult(ctx context.Context, req models.SmartTestResultRequest) (*models.SmartTestResultResponse, error) {
	facade.testResult = &req
	return &models.SmartTestResultResponse{Result: &models.SmartTestResultResponseResult{Result: "test-result"}}, nil
}

func (facade *fakeSmartCommandFacade) AttributeResult(ctx context.Context, req models.SmartAttributeResultRequest) (*models.SmartAttributeResultResponse, error) {
	facade.attributeReq = &req
	return &models.SmartAttributeResultResponse{Result: &models.SmartAttributeResultResponseResult{Result: "attributes"}}, nil
}

func (facade *fakeSmartCommandFacade) ExtendResult(ctx context.Context, req models.SmartExtendResultRequest) (*models.SmartExtendResultResponse, error) {
	facade.extendReq = &req
	return &models.SmartExtendResultResponse{Result: &models.SmartExtendResultResponseResult{Result: "extend"}}, nil
}

func TestSmartCommandWrappersDelegateParsedRequests(t *testing.T) {
	orig := newSmartCommandServiceFacade
	defer func() { newSmartCommandServiceFacade = orig }()

	facade := &fakeSmartCommandFacade{}
	newSmartCommandServiceFacade = func() smartCommandFacade { return facade }

	if _, err := SmartPostTest(context.Background(), jsonRequest(http.MethodPost, "/smart/test", `{"type":"short","devicePath":"/dev/sda"}`)); err != nil {
		t.Fatalf("smart test wrapper: %v", err)
	}
	if facade.testReq == nil || facade.testReq.Type != "short" || facade.testReq.DevicePath != "/dev/sda" {
		t.Fatalf("unexpected test request: %#v", facade.testReq)
	}

	if _, err := SmartPostTestResult(context.Background(), jsonRequest(http.MethodPost, "/smart/test-result", `{"type":"selftest","devicePath":"/dev/sdb"}`)); err != nil {
		t.Fatalf("smart test result wrapper: %v", err)
	}
	if facade.testResult == nil || facade.testResult.Type != "selftest" || facade.testResult.DevicePath != "/dev/sdb" {
		t.Fatalf("unexpected test result request: %#v", facade.testResult)
	}

	if _, err := SmartPostAttributeResult(context.Background(), jsonRequest(http.MethodPost, "/smart/attribute", `{"devicePath":"/dev/sdc"}`)); err != nil {
		t.Fatalf("smart attribute wrapper: %v", err)
	}
	if facade.attributeReq == nil || facade.attributeReq.DevicePath != "/dev/sdc" {
		t.Fatalf("unexpected attribute request: %#v", facade.attributeReq)
	}

	if _, err := SmartPostExtendResult(context.Background(), jsonRequest(http.MethodPost, "/smart/extend", `{"devicePath":"/dev/sdd"}`)); err != nil {
		t.Fatalf("smart extend wrapper: %v", err)
	}
	if facade.extendReq == nil || facade.extendReq.DevicePath != "/dev/sdd" {
		t.Fatalf("unexpected extend request: %#v", facade.extendReq)
	}
}
