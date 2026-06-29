package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildGuideSoftSourceByIndexPreservesCatalogOrder(t *testing.T) {
	t.Parallel()

	if len(guideSoftSourceIdentities) != 8 {
		t.Fatalf("unexpected source count: %d", len(guideSoftSourceIdentities))
	}

	source := buildGuideSoftSourceByIndex(guideSoftSourceIdentities[0], 0)
	if source.Identity != "OpenWrtHttp" || source.Name != "OpenWRT(HTTP)" || source.URL != "http://downloads.openwrt.org/" {
		t.Fatalf("unexpected first source: %#v", source)
	}

	last := buildGuideSoftSourceByIndex(guideSoftSourceIdentities[7], 7)
	if last.Identity != "Tencent Cloud" || last.Name != "腾讯云" || last.URL != "https://mirrors.cloud.tencent.com/openwrt/" {
		t.Fatalf("unexpected last source: %#v", last)
	}
}

func TestResolveGuideSoftSourceByIdentityReturnsKnownSource(t *testing.T) {
	t.Parallel()

	source := resolveGuideSoftSourceByIdentity("USTC")
	if source.Identity != "USTC" || source.Name != "中国科学技术大学" || source.URL != "https://mirrors.ustc.edu.cn/openwrt/" {
		t.Fatalf("unexpected identity mapping: %#v", source)
	}
}

func TestResolveGuideSoftSourceByIdentityReturnsEmptyForUnknownSource(t *testing.T) {
	t.Parallel()

	source := resolveGuideSoftSourceByIdentity("unknown")
	if source.Identity != "" || source.Name != "" || source.URL != "" {
		t.Fatalf("unexpected unknown identity mapping: %#v", source)
	}
}

func TestResolveGuideSoftSourceByURLReturnsKnownSource(t *testing.T) {
	t.Parallel()

	source := resolveGuideSoftSourceByURL("https://mirrors.aliyun.com/openwrt/")
	if source.Identity != "Alibaba Cloud" || source.Name != "阿里云" || source.URL != "https://mirrors.aliyun.com/openwrt/" {
		t.Fatalf("unexpected url mapping: %#v", source)
	}
}

func TestResolveGuideSoftSourceByURLFallsBackToRawURL(t *testing.T) {
	t.Parallel()

	source := resolveGuideSoftSourceByURL("https://example.com/openwrt/")
	if source.Identity != "https://example.com/openwrt/" || source.Name != "https://example.com/openwrt/" || source.URL != "https://example.com/openwrt/" {
		t.Fatalf("unexpected fallback mapping: %#v", source)
	}
}

func TestDefaultGuideSoftSourceReaderListSourcesPreservesCatalog(t *testing.T) {
	t.Parallel()

	reader := newDefaultGuideSoftSourceReader()
	list, err := reader.ListSources(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(list) != len(guideSoftSourceIdentities) {
		t.Fatalf("unexpected source count: %d", len(list))
	}
	if list[0].Identity != "OpenWrtHttp" || list[7].Identity != "Tencent Cloud" {
		t.Fatalf("unexpected list order: %#v %#v", list[0], list[7])
	}
}

func TestReadGuideSoftSourceFeedURLByContent(t *testing.T) {
	t.Parallel()

	url, err := readGuideSoftSourceFeedURLByContent("src/gz istoreos_base https://mirrors.aliyun.com/openwrt/packages/base\n")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if url != "https://mirrors.aliyun.com/openwrt/" {
		t.Fatalf("unexpected feed url: %q", url)
	}
}

func TestReadGuideSoftSourceFeedURLByContentReturnsLegacyErrorWhenMissing(t *testing.T) {
	t.Parallel()

	_, err := readGuideSoftSourceFeedURLByContent("src/gz something else\n")
	if err == nil || err.Error() != "feed not found" {
		t.Fatalf("unexpected missing feed error: %v", err)
	}
}

func TestDefaultGuideSoftSourceReaderReadCurrentSourceMapsOpenWrtHost(t *testing.T) {
	prevRead := readGuideSoftSourceFile
	defer func() { readGuideSoftSourceFile = prevRead }()
	readGuideSoftSourceFile = func(path string) ([]byte, error) {
		return []byte("src/gz istoreos_base https://downloads.openwrt.org/packages/base\n"), nil
	}

	reader := newDefaultGuideSoftSourceReader()
	source, err := reader.ReadCurrentSource(context.Background())
	if err != nil {
		t.Fatalf("unexpected current-source error: %v", err)
	}
	if source.Identity != "OpenWrtHttps" || source.URL != "https://downloads.openwrt.org/" {
		t.Fatalf("unexpected current source: %#v", source)
	}
}

func TestDefaultGuideSoftSourceReaderReadCurrentSourceMapsMirrorHost(t *testing.T) {
	prevRead := readGuideSoftSourceFile
	defer func() { readGuideSoftSourceFile = prevRead }()
	readGuideSoftSourceFile = func(path string) ([]byte, error) {
		return []byte("src/gz istoreos_base https://mirrors.sustech.edu.cn/packages/base\n"), nil
	}

	reader := newDefaultGuideSoftSourceReader()
	source, err := reader.ReadCurrentSource(context.Background())
	if err != nil {
		t.Fatalf("unexpected current-source error: %v", err)
	}
	if source.Identity != "SUSTech" || source.URL != "https://mirrors.sustech.edu.cn/openwrt/" {
		t.Fatalf("unexpected mirror source: %#v", source)
	}
}

func TestDefaultGuideSoftSourceWriterReplaceSourceFallsBackToTargetAndPreservesMode(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "rom-distfeeds.conf")
	targetFile := filepath.Join(tempDir, "distfeeds.conf")
	if err := os.WriteFile(targetFile, []byte("src/gz istoreos_base https://old.example.com/openwrt/packages/base\n"), 0600); err != nil {
		t.Fatalf("unexpected target write error: %v", err)
	}

	writer := &defaultGuideSoftSourceWriter{sourcePath: sourceFile, targetPath: targetFile}
	err := writer.ReplaceSource(context.Background(), resolveGuideSoftSourceByIdentity("Alibaba Cloud"))
	if err != nil {
		t.Fatalf("unexpected replace error: %v", err)
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("unexpected target read error: %v", err)
	}
	if !strings.Contains(string(content), "https://mirrors.aliyun.com/openwrt/") {
		t.Fatalf("unexpected replaced content: %s", string(content))
	}
	fi, err := os.Stat(targetFile)
	if err != nil {
		t.Fatalf("unexpected target stat error: %v", err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Fatalf("unexpected preserved mode: %v", fi.Mode().Perm())
	}
}
