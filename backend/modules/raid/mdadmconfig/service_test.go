package mdadmconfig

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeStore struct {
	arrays       []string
	scanOutput   string
	uuidOutput   string
	freeMd       []int
	loadErr      error
	deleteErr    error
	scanErr      error
	uuidErr      error
	addErr       error
	nextFreeCall int
	events       []string
}

func (store *fakeStore) LoadConfig(ctx context.Context) error {
	store.events = append(store.events, "load")
	return store.loadErr
}

func (store *fakeStore) Arrays(ctx context.Context) []string {
	store.events = append(store.events, "arrays")
	return store.arrays
}

func (store *fakeStore) DeleteFirstArray(ctx context.Context) error {
	store.events = append(store.events, "delete")
	return store.deleteErr
}

func (store *fakeStore) Scan(ctx context.Context) (string, error) {
	store.events = append(store.events, "scan")
	return store.scanOutput, store.scanErr
}

func (store *fakeStore) DiscoverMemberUUIDs(ctx context.Context) (string, error) {
	store.events = append(store.events, "uuids")
	return store.uuidOutput, store.uuidErr
}

func (store *fakeStore) FindFreeMd(min int) int {
	store.events = append(store.events, "find")
	if store.nextFreeCall >= len(store.freeMd) {
		return -1
	}
	value := store.freeMd[store.nextFreeCall]
	store.nextFreeCall++
	return value
}

func (store *fakeStore) AddArray(ctx context.Context, device string, uuid string) error {
	store.events = append(store.events, "add:"+device+":"+uuid)
	return store.addErr
}

func (store *fakeStore) Commit(ctx context.Context) {
	store.events = append(store.events, "commit")
}

func (store *fakeStore) Enable(ctx context.Context) {
	store.events = append(store.events, "enable")
}

func (store *fakeStore) Restart(ctx context.Context) {
	store.events = append(store.events, "restart")
}

func TestGenerateRebuildsConfigFromMdadmScan(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		arrays: []string{"old0", "old1"},
		scanOutput: "ARRAY /dev/md1 metadata=1.2 UUID=uuid-1 name=one\n" +
			"ignored line\n" +
			"ARRAY /dev/md2 metadata=1.2 UUID=uuid-2 name=two\n",
	}

	if err := NewService(store).Generate(context.Background()); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	want := []string{
		"load",
		"arrays",
		"delete",
		"delete",
		"scan",
		"add:/dev/md1:uuid-1",
		"add:/dev/md2:uuid-2",
		"commit",
		"enable",
	}
	if !reflect.DeepEqual(store.events, want) {
		t.Fatalf("unexpected events:\nwant=%#v\ngot=%#v", want, store.events)
	}
}

func TestGenerateWrapsStoreErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		init func(*fakeStore)
		want string
	}{
		{name: "load", init: func(s *fakeStore) { s.loadErr = errors.New("failed") }, want: "获取mdadm配置文件失败"},
		{name: "delete", init: func(s *fakeStore) { s.arrays = []string{"old"}; s.deleteErr = errors.New("failed") }, want: "删除mdadm配置失败"},
		{name: "scan", init: func(s *fakeStore) { s.scanErr = errors.New("failed") }, want: "mdadm -Ds 命令出错"},
		{name: "add", init: func(s *fakeStore) {
			s.scanOutput = "ARRAY /dev/md1 metadata=1.2 UUID=uuid-1"
			s.addErr = errors.New("failed")
		}, want: "添加mdadm配置失败"},
	}

	for _, tc := range cases {
		store := &fakeStore{}
		tc.init(store)
		err := NewService(store).Generate(context.Background())
		if err == nil || err.Error() != tc.want {
			t.Fatalf("%s: unexpected error want=%q got=%v", tc.name, tc.want, err)
		}
	}
}

func TestAutoFixRebuildsConfigFromDiscoveredUUIDs(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		arrays:     []string{"old"},
		uuidOutput: "uuid-1\nuuid-2",
		freeMd:     []int{0, 2},
	}

	if err := NewService(store).AutoFix(context.Background()); err != nil {
		t.Fatalf("AutoFix returned error: %v", err)
	}

	want := []string{
		"uuids",
		"load",
		"arrays",
		"delete",
		"find",
		"add:/dev/md0:uuid-1",
		"find",
		"add:/dev/md2:uuid-2",
		"commit",
		"restart",
	}
	if !reflect.DeepEqual(store.events, want) {
		t.Fatalf("unexpected events:\nwant=%#v\ngot=%#v", want, store.events)
	}
}

func TestAutoFixStopsWhenNoMdDeviceIsAvailable(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		uuidOutput: "uuid-1\nuuid-2",
		freeMd:     []int{-1},
	}

	if err := NewService(store).AutoFix(context.Background()); err != nil {
		t.Fatalf("AutoFix returned error: %v", err)
	}

	want := []string{"uuids", "load", "arrays", "find", "commit", "restart"}
	if !reflect.DeepEqual(store.events, want) {
		t.Fatalf("unexpected events:\nwant=%#v\ngot=%#v", want, store.events)
	}
}

func TestAutoFixWrapsErrors(t *testing.T) {
	t.Parallel()

	uuidStore := &fakeStore{uuidErr: errors.New("failed")}
	err := NewService(uuidStore).AutoFix(context.Background())
	if err == nil || err.Error() != "获取raid分区信息失败" {
		t.Fatalf("unexpected uuid error: %v", err)
	}

	addStore := &fakeStore{
		uuidOutput: "uuid-1",
		freeMd:     []int{0},
		addErr:     errors.New("failed"),
	}
	err = NewService(addStore).AutoFix(context.Background())
	if err == nil || err.Error() != "添加mdadm配置失败" {
		t.Fatalf("unexpected add error: %v", err)
	}
}
