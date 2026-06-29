package service

import (
	"context"
	"errors"
	"testing"
)

func TestDefaultGuideDockerRuntimeWriterDelegatesStartAndStop(t *testing.T) {
	originalStart := runGuideDockerStart
	originalStop := runGuideDockerStop
	defer func() {
		runGuideDockerStart = originalStart
		runGuideDockerStop = originalStop
	}()

	startCalls := 0
	stopCalls := 0
	runGuideDockerStart = func(ctx context.Context) error {
		startCalls++
		return nil
	}
	runGuideDockerStop = func(ctx context.Context) error {
		stopCalls++
		return nil
	}

	writer := newDefaultGuideDockerRuntimeWriter()
	if err := writer.Start(context.Background()); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	if err := writer.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
	if startCalls != 1 || stopCalls != 1 {
		t.Fatalf("unexpected seam calls: start=%d stop=%d", startCalls, stopCalls)
	}
}

func TestDefaultGuideDockerRuntimeWriterPropagatesErrors(t *testing.T) {
	originalStart := runGuideDockerStart
	originalStop := runGuideDockerStop
	defer func() {
		runGuideDockerStart = originalStart
		runGuideDockerStop = originalStop
	}()

	startErr := errors.New("start failed")
	stopErr := errors.New("stop failed")
	runGuideDockerStart = func(ctx context.Context) error { return startErr }
	runGuideDockerStop = func(ctx context.Context) error { return stopErr }

	writer := newDefaultGuideDockerRuntimeWriter()
	if err := writer.Start(context.Background()); !errors.Is(err, startErr) {
		t.Fatalf("expected start error, got %v", err)
	}
	if err := writer.Stop(context.Background()); !errors.Is(err, stopErr) {
		t.Fatalf("expected stop error, got %v", err)
	}
}
