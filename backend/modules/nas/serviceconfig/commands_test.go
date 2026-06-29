package serviceconfig

import (
	"reflect"
	"testing"
)

func TestBuildSambaShareCommands(t *testing.T) {
	t.Parallel()

	got := BuildSambaShareCommands(1, SambaCreateInput{
		ShareName:   "share",
		RootPath:    "/mnt/data",
		Username:    "user",
		AllowLegacy: true,
	})

	want := []string{
		"uci add samba4 sambashare",
		"uci set samba4.@samba[0].enabled=1",
		"uci set samba4.@samba[0].macos=1",
		"uci set samba4.@sambashare[1].name=share",
		"uci set samba4.@sambashare[1].path=/mnt/data",
		"uci set samba4.@sambashare[1].read_only=no",
		"uci set samba4.@sambashare[1].users=user",
		"uci set samba4.@sambashare[1].create_mask=0777",
		"uci set samba4.@sambashare[1].force_root=1",
		"uci set samba4.@samba[0].allow_legacy_protocols=1",
		"uci commit samba4",
		"/etc/init.d/samba4 restart",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected samba share commands:\nwant=%#v\ngot=%#v", want, got)
	}
}

func TestBuildSambaShareCommandsOmitsLegacyProtocolWhenDisabled(t *testing.T) {
	t.Parallel()

	got := BuildSambaShareCommands(0, SambaCreateInput{
		ShareName: "share",
		RootPath:  "/mnt/data",
		Username:  "user",
	})

	want := []string{
		"uci add samba4 sambashare",
		"uci set samba4.@samba[0].enabled=1",
		"uci set samba4.@samba[0].macos=1",
		"uci set samba4.@sambashare[0].name=share",
		"uci set samba4.@sambashare[0].path=/mnt/data",
		"uci set samba4.@sambashare[0].read_only=no",
		"uci set samba4.@sambashare[0].users=user",
		"uci set samba4.@sambashare[0].create_mask=0777",
		"uci set samba4.@sambashare[0].force_root=1",
		"uci commit samba4",
		"/etc/init.d/samba4 restart",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected samba share commands without legacy:\nwant=%#v\ngot=%#v", want, got)
	}
}

func TestBuildWebdavConfigCommands(t *testing.T) {
	t.Parallel()

	got := BuildWebdavConfigCommands(WebdavCreateInput{
		RootPath: "/mnt/data",
		Username: "user",
		Password: "pw",
	})

	want := []string{
		"uci set gowebdav.config.root_dir=/mnt/data",
		"uci set gowebdav.config.enable=1",
		"uci set gowebdav.config.username=user",
		"uci set gowebdav.config.password=pw",
		"uci set gowebdav.config.allow_wan=1",
		"uci commit gowebdav",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected WebDAV config commands:\nwant=%#v\ngot=%#v", want, got)
	}
}
