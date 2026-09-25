package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/modules/lancontrol/staticassignment"
)

var lanStaticAssignmentWriteTestMu sync.Mutex

func withStaticAssignmentWriteSeams(t *testing.T) {
	t.Helper()
	originalLoad := lanStaticAssignmentWriteLoadConfig
	originalSections := lanStaticAssignmentWriteGetSections
	originalGetLast := lanStaticAssignmentWriteGetLast
	originalSnapshot := lanStaticAssignmentWriteSnapshot
	originalPreflight := lanStaticAssignmentWritePreflight
	originalMutate := lanStaticAssignmentWriteMutate
	originalReload := lanStaticAssignmentWriteReload
	originalRestore := lanStaticAssignmentWriteRestore
	t.Cleanup(func() {
		lanStaticAssignmentWriteLoadConfig = originalLoad
		lanStaticAssignmentWriteGetSections = originalSections
		lanStaticAssignmentWriteGetLast = originalGetLast
		lanStaticAssignmentWriteSnapshot = originalSnapshot
		lanStaticAssignmentWritePreflight = originalPreflight
		lanStaticAssignmentWriteMutate = originalMutate
		lanStaticAssignmentWriteReload = originalReload
		lanStaticAssignmentWriteRestore = originalRestore
	})
}

func configureStaticAssignmentHappyPath(t *testing.T) *[]string {
	t.Helper()
	calls := &[]string{}
	lanStaticAssignmentWriteLoadConfig = func(string, bool) error { *calls = append(*calls, "load"); return nil }
	lanStaticAssignmentWriteGetSections = func(string, string) ([]string, bool) { return nil, true }
	lanStaticAssignmentWriteGetLast = func(string, string, string) (string, bool) { return "", false }
	lanStaticAssignmentWritePreflight = func(context.Context, StaticAssignmentWriteInput) error {
		*calls = append(*calls, "preflight")
		return nil
	}
	lanStaticAssignmentWriteSnapshot = func() (dhcpConfigSnapshot, error) {
		*calls = append(*calls, "snapshot")
		return dhcpConfigSnapshot{Data: []byte("old"), Mode: 0o600}, nil
	}
	lanStaticAssignmentWriteMutate = func(StaticAssignmentWriteInput, []staticassignment.HostRecord) error {
		*calls = append(*calls, "mutate")
		return nil
	}
	lanStaticAssignmentWriteReload = func(context.Context) error { *calls = append(*calls, "reload"); return nil }
	lanStaticAssignmentWriteRestore = func(context.Context, dhcpConfigSnapshot) error { *calls = append(*calls, "restore"); return nil }
	return calls
}

func TestDefaultStaticAssignmentWriteStoreAppliesSafeTransaction(t *testing.T) {
	lanStaticAssignmentWriteTestMu.Lock()
	defer lanStaticAssignmentWriteTestMu.Unlock()
	withStaticAssignmentWriteSeams(t)
	calls := configureStaticAssignmentHappyPath(t)
	var gotInput StaticAssignmentWriteInput
	lanStaticAssignmentWriteMutate = func(input StaticAssignmentWriteInput, _ []staticassignment.HostRecord) error {
		*calls = append(*calls, "mutate")
		gotInput = input
		return nil
	}
	err := (&defaultStaticAssignmentWriteStore{}).ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:03", Hostname: "printer",
	})
	if err != nil {
		t.Fatalf("ApplyStaticAssignment: %v", err)
	}
	if strings.Join(*calls, ",") != "load,preflight,snapshot,mutate,reload" {
		t.Fatalf("calls = %v", *calls)
	}
	if gotInput.Hostname != "printer" {
		t.Fatalf("mutate input = %+v", gotInput)
	}
}

func TestDefaultStaticAssignmentWriteStoreRejectsConflictBeforeMutation(t *testing.T) {
	lanStaticAssignmentWriteTestMu.Lock()
	defer lanStaticAssignmentWriteTestMu.Unlock()
	withStaticAssignmentWriteSeams(t)
	calls := configureStaticAssignmentHappyPath(t)
	lanStaticAssignmentWriteGetSections = func(string, string) ([]string, bool) { return []string{"cfg01"}, true }
	lanStaticAssignmentWriteGetLast = func(_, _, option string) (string, bool) {
		if option == "mac" {
			return "AA:BB:CC:DD:EE:99", true
		}
		if option == "ip" {
			return "192.168.100.20", true
		}
		return "", false
	}
	err := (&defaultStaticAssignmentWriteStore{}).ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:03", AssignedIP: "192.168.100.20", BindIP: true,
	})
	if err == nil || err.Error() != "ip is already in use" {
		t.Fatalf("err = %v", err)
	}
	if strings.Join(*calls, ",") != "load" {
		t.Fatalf("unexpected calls = %v", *calls)
	}
}

func TestDefaultStaticAssignmentWriteStoreStopsOnPreflightFailure(t *testing.T) {
	lanStaticAssignmentWriteTestMu.Lock()
	defer lanStaticAssignmentWriteTestMu.Unlock()
	withStaticAssignmentWriteSeams(t)
	calls := configureStaticAssignmentHappyPath(t)
	lanStaticAssignmentWritePreflight = func(context.Context, StaticAssignmentWriteInput) error {
		*calls = append(*calls, "preflight")
		return errors.New("bad DHCP host name")
	}
	err := (&defaultStaticAssignmentWriteStore{}).ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:03"})
	if err == nil || !strings.Contains(err.Error(), "preflight") {
		t.Fatalf("err = %v", err)
	}
	if strings.Join(*calls, ",") != "load,preflight" {
		t.Fatalf("unexpected calls = %v", *calls)
	}
}

func TestDefaultStaticAssignmentWriteStoreRollsBackMutationAndReloadFailures(t *testing.T) {
	for _, stage := range []string{"mutate", "reload"} {
		stage := stage
		t.Run(stage, func(t *testing.T) {
			lanStaticAssignmentWriteTestMu.Lock()
			defer lanStaticAssignmentWriteTestMu.Unlock()
			withStaticAssignmentWriteSeams(t)
			calls := configureStaticAssignmentHappyPath(t)
			if stage == "mutate" {
				lanStaticAssignmentWriteMutate = func(StaticAssignmentWriteInput, []staticassignment.HostRecord) error {
					*calls = append(*calls, "mutate")
					return errors.New("write failed")
				}
			} else {
				lanStaticAssignmentWriteReload = func(context.Context) error { *calls = append(*calls, "reload"); return errors.New("dnsmasq stopped") }
			}
			err := (&defaultStaticAssignmentWriteStore{}).ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:03"})
			if err == nil {
				t.Fatal("expected error")
			}
			if (*calls)[len(*calls)-1] != "restore" {
				t.Fatalf("rollback was not last call: %v", *calls)
			}
		})
	}
}

func TestMutateStaticAssignmentConfigAtWritesTypedUCIWithoutShell(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initial := "config dnsmasq 'main'\n\toption domainneeded '1'\n\nconfig host 'old_host'\n\toption mac 'AA:BB:CC:DD:EE:03'\n\toption name 'old'\n"
	if err := os.WriteFile(filepath.Join(dir, "dhcp"), []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	input := StaticAssignmentWriteInput{
		Action: "add", AssignedMAC: "AA:BB:CC:DD:EE:03", AssignedIP: "192.168.100.20", BindIP: true,
		Hostname: "living-room-tv", TagName: "t_auto_c0a86401", TagTitle: "旁路由", MaterializeAutoTag: true,
	}
	hosts := []staticassignment.HostRecord{{SectionName: "old_host", MAC: input.AssignedMAC}}
	if err := mutateStaticAssignmentConfigAt(dir, input, hosts); err != nil {
		t.Fatalf("mutate config: %v", err)
	}
	tree := uci.NewTree(dir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	section := "quickstart_aabbccddee03"
	if got, _ := tree.GetLast("dhcp", section, "name"); got != "living-room-tv" {
		t.Fatalf("name = %q", got)
	}
	if got, _ := tree.GetLast("dhcp", section, "tag_title"); got != "旁路由" {
		t.Fatalf("tag title = %q", got)
	}
	if got, _ := tree.GetLast("dhcp", section, "ip"); got != "192.168.100.20" {
		t.Fatalf("ip = %q", got)
	}
	if sections, _ := tree.GetSections("dhcp", "host"); len(sections) != 1 || sections[0] != section {
		t.Fatalf("host sections = %v", sections)
	}
}
