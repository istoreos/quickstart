package linkease

import (
	"context"
	"errors"
	"testing"
)

type fakeConfigReader struct {
	enabled string
	port    string
	err     error
}

func (reader *fakeConfigReader) ReadEnabled(ctx context.Context) (string, error) {
	if reader.err != nil {
		return "", reader.err
	}
	return reader.enabled, nil
}

func (reader *fakeConfigReader) ReadPort(ctx context.Context) (string, error) {
	if reader.err != nil {
		return "", reader.err
	}
	return reader.port, nil
}

type fakeConfigWriter struct {
	calls int
	err   error
}

func (writer *fakeConfigWriter) Enable(ctx context.Context) error {
	writer.calls++
	return writer.err
}

func TestServiceSkipsWriterWhenAlreadyEnabled(t *testing.T) {
	t.Parallel()

	writer := &fakeConfigWriter{}
	svc := NewService(&fakeConfigReader{
		enabled: "1",
		port:    "8897",
	}, writer)

	result, err := svc.Enable(context.Background())
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if writer.calls != 0 {
		t.Fatalf("expected no writer calls when already enabled, got %d", writer.calls)
	}
	if result == nil || result.Port != "8897" {
		t.Fatalf("unexpected service result: %#v", result)
	}
}

func TestServiceEnablesWhenDisabled(t *testing.T) {
	t.Parallel()

	writer := &fakeConfigWriter{}
	svc := NewService(&fakeConfigReader{
		enabled: "0",
		port:    "8897",
	}, writer)

	result, err := svc.Enable(context.Background())
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if writer.calls != 1 {
		t.Fatalf("expected writer to be called once, got %d", writer.calls)
	}
	if result == nil || result.Port != "8897" {
		t.Fatalf("unexpected service result: %#v", result)
	}
}

func TestServicePropagatesErrors(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read enabled failed")
	svc := NewService(&fakeConfigReader{err: readErr}, &fakeConfigWriter{})
	if _, err := svc.Enable(context.Background()); !errors.Is(err, readErr) {
		t.Fatalf("expected read error, got %v", err)
	}

	writeErr := errors.New("enable failed")
	svc = NewService(&fakeConfigReader{
		enabled: "0",
		port:    "8897",
	}, &fakeConfigWriter{err: writeErr})
	if _, err := svc.Enable(context.Background()); !errors.Is(err, writeErr) {
		t.Fatalf("expected writer error, got %v", err)
	}
}
