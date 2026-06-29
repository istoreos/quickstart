package writecommands

import (
	"reflect"
	"testing"
)

func TestNormalizeLevel(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"linear": "linear",
		"raid0":  "0",
		"raid1":  "1",
		"raid5":  "5",
		"raid6":  "6",
		"raid10": "10",
		"other":  "",
	}
	for input, want := range cases {
		if got := NormalizeLevel(input); got != want {
			t.Fatalf("NormalizeLevel(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestValidateMemberCount(t *testing.T) {
	t.Parallel()

	valid := []struct {
		level string
		count int
	}{
		{level: "linear", count: 2},
		{level: "0", count: 2},
		{level: "1", count: 2},
		{level: "5", count: 3},
		{level: "6", count: 3},
		{level: "10", count: 4},
	}
	for _, tc := range valid {
		if err := ValidateMemberCount(tc.level, tc.count); err != nil {
			t.Fatalf("ValidateMemberCount(%q, %d) returned error: %v", tc.level, tc.count, err)
		}
	}

	invalid := []struct {
		level string
		count int
	}{
		{level: "linear", count: 1},
		{level: "5", count: 2},
		{level: "6", count: 2},
		{level: "10", count: 3},
	}
	for _, tc := range invalid {
		if err := ValidateMemberCount(tc.level, tc.count); err == nil || err.Error() != "没有足够的成员设备" {
			t.Fatalf("ValidateMemberCount(%q, %d) error=%v, want member error", tc.level, tc.count, err)
		}
	}
}

func TestBuildCreateCommand(t *testing.T) {
	t.Parallel()

	got := BuildCreateCommand("/dev/md1", "1", []string{"/dev/sda1", "/dev/sdb1"})
	want := "mdadm -C /dev/md1 --run --quiet --assume-clean --homehost=any -n 2 -l 1 /dev/sda1 /dev/sdb1"
	if got != want {
		t.Fatalf("unexpected create command:\nwant=%q\ngot=%q", want, got)
	}
}

func TestBuildDeleteCommands(t *testing.T) {
	t.Parallel()

	got := BuildDeleteCommands("/dev/md1", []string{"/dev/sda1", "/dev/sdb1"})
	want := []string{
		"mdadm --stop /dev/md1",
		"mdadm --remove /dev/md1",
		"mdadm --zero-superblock /dev/sda1 /dev/sdb1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected delete commands:\nwant=%#v\ngot=%#v", want, got)
	}
}

func TestBuildAddGrowRemoveRecoverCommands(t *testing.T) {
	t.Parallel()

	if got, want := BuildAddCommand("/dev/md1", "/dev/sdc1"), "mdadm -a /dev/md1 /dev/sdc1"; got != want {
		t.Fatalf("unexpected add command: want=%q got=%q", want, got)
	}
	if got, want := BuildGrowCommand("/dev/md1", 3), "mdadm -G /dev/md1 -n 3"; got != want {
		t.Fatalf("unexpected grow command: want=%q got=%q", want, got)
	}

	remove := BuildRemoveCommands("/dev/md1", "/dev/sdc1")
	wantRemove := []string{
		"mdadm --manage /dev/md1 --fail /dev/sdc1",
		"mdadm --manage /dev/md1 --remove /dev/sdc1",
	}
	if !reflect.DeepEqual(remove, wantRemove) {
		t.Fatalf("unexpected remove commands:\nwant=%#v\ngot=%#v", wantRemove, remove)
	}

	recover := BuildRecoverCommands("/dev/md1", "/dev/sdc1")
	wantRecover := []string{"mdadm -a /dev/md1 /dev/sdc1"}
	if !reflect.DeepEqual(recover, wantRecover) {
		t.Fatalf("unexpected recover commands:\nwant=%#v\ngot=%#v", wantRecover, recover)
	}
}

func TestBuildRaidPartitionCommand(t *testing.T) {
	t.Parallel()

	got, err := BuildRaidPartitionCommand("/dev/sda")
	if err != nil {
		t.Fatalf("BuildRaidPartitionCommand returned error: %v", err)
	}
	want := "parted -a opt /dev/sda mkpart primary ext4 16M 100% set 1 raid on"
	if got != want {
		t.Fatalf("unexpected raid partition command:\nwant=%q\ngot=%q", want, got)
	}

	if _, err := BuildRaidPartitionCommand(""); err == nil || err.Error() != "param missing" {
		t.Fatalf("unexpected empty device error: %v", err)
	}
}

func TestBuildMdadmConfigCommands(t *testing.T) {
	t.Parallel()

	got := BuildMdadmArrayCommands("/dev/md1", "uuid-1")
	want := []string{
		"uci add mdadm array",
		"uci set mdadm.@array[-1].device=/dev/md1",
		"uci set mdadm.@array[-1].uuid=uuid-1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected mdadm array commands:\nwant=%#v\ngot=%#v", want, got)
	}

	if got, want := BuildDeleteFirstMdadmArrayCommand(), []string{"uci del mdadm.@array[0]"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected mdadm delete command:\nwant=%#v\ngot=%#v", want, got)
	}
	if got, want := BuildCommitMdadmCommand(), []string{"uci commit mdadm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected mdadm commit command:\nwant=%#v\ngot=%#v", want, got)
	}
	if got, want := BuildRestartMdadmCommand(), []string{"/etc/init.d/mdadm restart"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected mdadm restart command:\nwant=%#v\ngot=%#v", want, got)
	}
	if got, want := BuildEnableMdadmCommand(), []string{"/etc/init.d/mdadm enable"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected mdadm enable command:\nwant=%#v\ngot=%#v", want, got)
	}
}

func TestBuildAutoFixUUIDCommand(t *testing.T) {
	t.Parallel()

	got := BuildAutoFixUUIDCommand()
	want := `block info | grep -Fw 'TYPE="linux_raid_member"' | sed -n 's/^.* UUID="\([a-f0-9-]\+\)".*$/\1/igp' | sort -u | sed 's/\([a-f0-9]\{8\}\)-\([a-f0-9]\{4\}\)-\([a-f0-9]\{4\}\)-\([a-f0-9]\{4\}\)-\([a-f0-9]\{4\}\)\([a-f0-9]\{8\}\)/\1:\2\3:\4\5:\6/ig'`
	if got != want {
		t.Fatalf("unexpected auto fix uuid command:\nwant=%q\ngot=%q", want, got)
	}
}
