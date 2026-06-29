package diskcommands

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeRunner struct {
	runCalls    [][]string
	outErrCalls [][]string
	outputCalls []string

	runErr    error
	outErrOut string
	outErrErr string
	outErrRun error
	outputErr error
}

func (runner *fakeRunner) Run(ctx context.Context, commands []string) error {
	runner.runCalls = append(runner.runCalls, append([]string(nil), commands...))
	return runner.runErr
}

func (runner *fakeRunner) OutErr(ctx context.Context, commands []string) (string, string, error) {
	runner.outErrCalls = append(runner.outErrCalls, append([]string(nil), commands...))
	return runner.outErrOut, runner.outErrErr, runner.outErrRun
}

func (runner *fakeRunner) Output(ctx context.Context, command string) ([]byte, error) {
	runner.outputCalls = append(runner.outputCalls, command)
	return nil, runner.outputErr
}

func TestMountRunsBusyboxMountCommands(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	svc := NewService(runner)

	if err := svc.Mount(context.Background(), "/dev/sda1", "/mnt/data"); err != nil {
		t.Fatalf("Mount returned error: %v", err)
	}

	want := [][]string{{
		"test -d '/mnt/data' || mkdir -p '/mnt/data'",
		"busybox mount '/dev/sda1' '/mnt/data'",
	}}
	if !reflect.DeepEqual(runner.outErrCalls, want) {
		t.Fatalf("unexpected mount commands:\nwant=%#v\ngot=%#v", want, runner.outErrCalls)
	}
}

func TestMountValidatesInputAndFormatsError(t *testing.T) {
	t.Parallel()

	if err := NewService(&fakeRunner{}).Mount(context.Background(), "", "/mnt/data"); err == nil {
		t.Fatal("expected missing device error")
	}

	runner := &fakeRunner{outErrErr: "stderr", outErrRun: errors.New("failed")}
	err := NewService(runner).Mount(context.Background(), "/dev/sda1", "/mnt/data")
	if err == nil || err.Error() != "mount '/dev/sda1' '/mnt/data' failed: stderr" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnmountVariants(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	svc := NewService(runner)

	if err := svc.UnMount(context.Background(), "/dev/sda1"); err != nil {
		t.Fatalf("UnMount returned error: %v", err)
	}
	if err := svc.Unmount(context.Background(), "-"); err != nil {
		t.Fatalf("Unmount '-' returned error: %v", err)
	}
	if err := svc.Unmount(context.Background(), "/overlay"); err == nil {
		t.Fatal("expected overlay unmount to fail")
	}
	if err := svc.Unmount(context.Background(), "/mnt/data"); err != nil {
		t.Fatalf("Unmount returned error: %v", err)
	}

	want := [][]string{
		{"umount '/dev/sda1'"},
		{"umount '/mnt/data'"},
	}
	if !reflect.DeepEqual(runner.outErrCalls, want) {
		t.Fatalf("unexpected unmount commands:\nwant=%#v\ngot=%#v", want, runner.outErrCalls)
	}
}

func TestDiskMutationCommands(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	svc := NewService(runner)

	if err := svc.Erase(context.Background(), "/dev/sda"); err != nil {
		t.Fatalf("Erase returned error: %v", err)
	}
	if err := svc.MakePart(context.Background(), "/dev/sda"); err != nil {
		t.Fatalf("MakePart returned error: %v", err)
	}
	if err := svc.Ext4Partition(context.Background(), "/dev/sda1"); err != nil {
		t.Fatalf("Ext4Partition returned error: %v", err)
	}

	wantRun := [][]string{{
		"dd if=/dev/zero of=/dev/sda bs=4096 count=8192",
		"parted -a optimal -s /dev/sda mklabel gpt",
	}}
	if !reflect.DeepEqual(runner.runCalls, wantRun) {
		t.Fatalf("unexpected run commands:\nwant=%#v\ngot=%#v", wantRun, runner.runCalls)
	}
	wantOutput := []string{
		"parted -a opt /dev/sda mkpart primary ext4 16M 100%",
		"mkfs.ext4 -F /dev/sda1",
	}
	if !reflect.DeepEqual(runner.outputCalls, wantOutput) {
		t.Fatalf("unexpected output commands:\nwant=%#v\ngot=%#v", wantOutput, runner.outputCalls)
	}
}

func TestAddFstabCreatesNewMount(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	got, err := NewService(runner).AddFstab(context.Background(), "uuid-1", "/mnt/data", false, nil)
	if err != nil {
		t.Fatalf("AddFstab returned error: %v", err)
	}
	if got != "/mnt/data" {
		t.Fatalf("unexpected target: %q", got)
	}

	want := [][]string{{
		"uci add fstab mount",
		"uci set fstab.@mount[-1].uuid=uuid-1",
		"uci set 'fstab.@mount[-1].target=/mnt/data'",
		"uci set fstab.@mount[-1].enabled=1",
	}}
	if !reflect.DeepEqual(runner.runCalls, want) {
		t.Fatalf("unexpected add fstab commands:\nwant=%#v\ngot=%#v", want, runner.runCalls)
	}
}

func TestAddFstabUpdatesExistingMount(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	_, err := NewService(runner).AddFstab(context.Background(), "uuid-1", "/mnt/new", false, []FstabMount{
		{Name: "cfg123", UUID: "uuid-1", Target: "/mnt/old"},
	})
	if err != nil {
		t.Fatalf("AddFstab returned error: %v", err)
	}

	want := [][]string{{
		"uci set fstab.cfg123.uuid=uuid-1",
		"uci set 'fstab.cfg123.target=/mnt/new'",
		"uci set fstab.cfg123.enabled=1",
	}}
	if !reflect.DeepEqual(runner.runCalls, want) {
		t.Fatalf("unexpected update fstab commands:\nwant=%#v\ngot=%#v", want, runner.runCalls)
	}
}

func TestAddFstabSkipExistingReturnsExistingTarget(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	got, err := NewService(runner).AddFstab(context.Background(), "uuid-1", "/mnt/new", true, []FstabMount{
		{Name: "cfg123", UUID: "uuid-1", Target: "/mnt/old"},
	})
	if err != nil {
		t.Fatalf("AddFstab returned error: %v", err)
	}
	if got != "/mnt/old" {
		t.Fatalf("unexpected existing target: %q", got)
	}
	if len(runner.runCalls) != 0 {
		t.Fatalf("did not expect fstab commands, got %#v", runner.runCalls)
	}
}

func TestCommitFstabCommands(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	svc := NewService(runner)

	if err := svc.CommitFstab(context.Background()); err != nil {
		t.Fatalf("CommitFstab returned error: %v", err)
	}
	if err := svc.CommitFstabAndBlockMount(context.Background()); err != nil {
		t.Fatalf("CommitFstabAndBlockMount returned error: %v", err)
	}

	want := [][]string{
		{"uci commit fstab"},
		{"uci commit fstab", "block mount"},
	}
	if !reflect.DeepEqual(runner.runCalls, want) {
		t.Fatalf("unexpected commit commands:\nwant=%#v\ngot=%#v", want, runner.runCalls)
	}
}
