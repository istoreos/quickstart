#!/usr/bin/env python3
"""Generate a test-only Go overlay from the pinned pre-multi-LAN provider.
Run inside the SSD build-cache environment. Product source is never copied.
"""
import hashlib
import json
import os
import re
import subprocess
from pathlib import Path

commit = "66589b22f140615db97851b6c3b1eec7aad975d5"
root = Path(__file__).resolve().parents[2]
raw = subprocess.check_output(["git", "-C", str(root), "show", commit + ":backend/service/rate_limit_native_provider.go"])
source = raw.decode()
identifier = r"\b(?:native[A-Za-z0-9_]*|NewNativeRateLimitProvider|desiredNativeRate|bitsPerSecondToMbit|defaultNativePolicyBaseURL)\b"
# Protect literals: these identify the original protocol and must not change.
token = r'"(?:\\.|[^"\\])*"|`[^`]*`|' + identifier
source = re.sub(token, lambda match: match[0] if match[0].startswith(('"', '`')) else "legacyContract" + match[0][0].upper() + match[0][1:], source)
source = source.replace('"time"', '"time"\n "testing"')
source += r'''
func TestMultiLANLegacyClientLiveBinding(t *testing.T) {
 base := os.Getenv("QUICKSTART_MULTI_LAN_LEGACY_BASE_URL")
 if base == "" { t.Skip("isolated compatibility engine required") }
 if !strings.HasPrefix(base, "http://127.0.0.1:") { t.Fatal("loopback test endpoint required") }
 provider := legacyContractNewNativeRateLimitProvider(base, nil)
 target := rateLimitTarget{MAC: "02:00:00:00:00:77"}
 if err := provider.Apply(context.Background(), target, models.DeviceSpeedPolicy{Enabled:true, UploadSpeed:3, DownloadSpeed:7}); err != nil { t.Fatal(err) }
 observed, err := provider.Inspect(context.Background(),target)
 if err != nil || !observed.Verified || observed.Policy.UploadSpeed !=3 { t.Fatalf("observed=%+v err=%v",observed,err) }
 if err := provider.Apply(context.Background(),target,models.DeviceSpeedPolicy{}); err != nil { t.Fatal(err) }
 t.Log("original provider from 66589b22 read/apply/verify/remove passed")
}
'''
assert 'Engine != "quickstart-native"' in source
work = Path(os.environ["BUILD_CACHE_WORK_DIR"]) / "multi-lan/M06"
work.mkdir(parents=True, exist_ok=True)
file = work / "legacy-provider_test.go"
file.write_text(source)
overlay = work / "legacy-overlay.json"
overlay.write_text(json.dumps({"Replace": {str(root / "backend/service/zz_multi_lan_legacy_provider_test.go"): str(file)}}))
(work / "legacy-provider-source.json").write_text(json.dumps({"commit": commit, "sourceSha256": hashlib.sha256(raw).hexdigest()}, indent=2) + "\n")
print(overlay)
