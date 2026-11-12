#!/usr/bin/env python3
"""Add a disposable baseline-only HTTP entrypoint to the pinned old SSD source."""
import hashlib,json,os,pathlib
work=pathlib.Path(os.environ['BUILD_CACHE_WORK_DIR'])/'multi-lan/M10/legacy'
backend=work/'backend'
if not (backend/'go.mod').is_file():raise SystemExit('extract pinned 66589b22 backend into SSD legacy directory first')
source=r'''package service
import "github.com/istoreos/quickstart/backend/models"
import("context";"encoding/json";"fmt";"net";"net/http";"os";"path/filepath";"strings";"testing";"time")
func TestMultiLANLegacyBaselineHTTP(t *testing.T){
 base:=os.Getenv("QUICKSTART_MULTI_LAN_CAPACITY_BASE");if base==""{t.Skip("explicit disposable baseline required")}
 marker,err:=os.ReadFile(filepath.Join(base,"owner"));current,_:=os.Readlink("/proc/self/ns/net");initial,_:=os.Readlink("/proc/1/ns/net")
 if err!=nil||string(marker)!="M10 isolated capacity"||!strings.HasPrefix(base,"/tmp/quickstart-multi-lan/m10-")||current==initial{t.Fatal("unsafe legacy baseline")}
 observations:=[]deviceInventoryObservation{}
 for i:=1;i<=256;i++{observations=append(observations,deviceInventoryObservation{MAC:fmt.Sprintf("02:00:00:00:%02x:%02x",i>>8,i&255),IPv4:fmt.Sprintf("10.201.%d.%d",i/250,i%250+2),Online:true})}
 source:=&fakeDeviceInventorySource{snapshots:[]deviceInventorySourceSnapshot{{ARP:observations}}}
 inventory:=newDeviceInventoryModuleForTest(source,filepath.Join(base,"legacy-history.json"),"m10-legacy",time.Now,2048)
 inventory.cacheTTL=defaultDeviceInventoryCacheTTL
 backend:=&ServiceBackend{deviceInventory:inventory,networkAudit:NewNetworkAuditModule(filepath.Join(base,"legacy-audit.json"),nil)}
 listener,err:=net.Listen("tcp","127.0.0.1:19482");if err!=nil{t.Fatal(err)}
 server:=&http.Server{Handler:http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  if r.Method!="GET"||r.URL.Path!="/cgi-bin/luci/istore/lanctrl/v2/devices/"{http.NotFound(w,r);return}
  response,err:=backend.GetDeviceInventoryV2(context.Background());if err!=nil{http.Error(w,err.Error(),500);return};w.Header().Set("Content-Type","application/json");_ = json.NewEncoder(w).Encode(response)
 }),ReadHeaderTimeout:5*time.Second};go server.Serve(listener);defer server.Shutdown(context.Background())
 for{time.Sleep(time.Second);if _,err:=os.Stat(filepath.Join(base,"stop"));err==nil{return}}
}
'''
source += r'''
func TestMultiLANLegacyWriteUpgradeFixture(t *testing.T){
 base:=os.Getenv("QUICKSTART_MULTI_LAN_UPGRADE_BASE");if base==""{t.Skip("explicit upgrade fixture directory required")}
 if !filepath.IsAbs(base)||!strings.Contains(base,"/multi-lan/M10/legacy-data"){t.Fatal("unsafe upgrade fixture")};if err:=os.MkdirAll(base,0700);err!=nil{t.Fatal(err)}
 now:=time.Now().UTC();id:="mac:02:00:00:00:00:55"
 snapshot:=deviceInventorySourceSnapshot{ARP:[]deviceInventoryObservation{{MAC:"02:00:00:00:00:55",IPv4:"192.168.100.55",Online:true}}}
 inventory:=newDeviceInventoryModuleForTest(&fakeDeviceInventorySource{snapshots:[]deviceInventorySourceSnapshot{snapshot}},filepath.Join(base,"inventory.json"),"old-boot",func()time.Time{return now},2048)
 alias,brand,category,mode,icon:="旧版设备资料","ASUS","computer","manual","laptop"
 result,err:=NewDeviceProfileModule(inventory).Apply(context.Background(),&models.DeviceProfileApplyRequest{DeviceID:id,Action:"patch",Alias:&alias,Brand:&brand,Category:&category,IconMode:&mode,IconKey:&icon});if err!=nil||result.Result.Error!=nil{t.Fatal("old profile",err,result)}
 groups:=newJSONDeviceGroupStore(filepath.Join(base,"groups.json"))
 if _,err:=groups.Mutate(context.Background(),&models.DeviceGroupMutationRequest{Action:"upsert_group",Group:&models.DeviceGroup{ID:"family",Name:"已有分组",Members:[]string{id},Policy:&models.GroupPolicy{Speed:&models.GroupSpeedPolicy{Enabled:true,UploadSpeed:3,DownloadSpeed:6},Schedules:[]*models.GroupSchedule{{ID:"night",Enabled:true,Days:[]int{0,1,2,3,4,5,6},StartMinute:1380,EndMinute:60,Action:"block"}}}}},"");err!=nil{t.Fatal(err)}
 policyStore:=availablePolicyStore();policyStore.policy.DeviceID=id
 insights:=NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(base,"traffic.json")),newDevicePolicyModuleForTest(policyStore));insights.now=func()time.Time{return now}
 if _,err:=insights.SetQuota(context.Background(),&models.TrafficQuotaRequest{DeviceID:id,Enabled:true,Period:"weekly",LimitBytes:50000000,Action:"notify"});err!=nil{t.Fatal(err)}
 if err:=insights.Record(context.Background(),insightTraffic(id,100,200));err!=nil{t.Fatal(err)};now=now.Add(6*time.Minute)
 if err:=insights.Record(context.Background(),insightTraffic(id,1350,3950));err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(base,"seed-time"),[]byte(now.Format(time.RFC3339Nano)),0600);err!=nil{t.Fatal(err)}
}
func TestMultiLANLegacyReadUpgradedDefaultData(t *testing.T){
 base:=os.Getenv("QUICKSTART_MULTI_LAN_UPGRADE_BASE");if base==""{t.Skip("explicit upgrade fixture directory required")}
 raw,err:=os.ReadFile(filepath.Join(base,"seed-time"));if err!=nil{t.Fatal(err)};now,err:=time.Parse(time.RFC3339Nano,string(raw));if err!=nil{t.Fatal(err)}
 id:="mac:02:00:00:00:00:55";snapshot:=deviceInventorySourceSnapshot{ARP:[]deviceInventoryObservation{{MAC:"02:00:00:00:00:55",IPv4:"192.168.100.55",Online:true}}}
 inventory:=newDeviceInventoryModuleForTest(&fakeDeviceInventorySource{snapshots:[]deviceInventorySourceSnapshot{snapshot}},filepath.Join(base,"inventory.json"),"old-boot",func()time.Time{return now},2048)
 profile,err:=NewDeviceProfileModule(inventory).Get(context.Background(),id);if err!=nil||profile.Result.Profile.Alias!="旧版设备资料"{t.Fatal("old reader lost profile",err,profile)}
 groups,err:=newJSONDeviceGroupStore(filepath.Join(base,"groups.json")).Read(context.Background());if err!=nil||len(groups.Groups)!=1{t.Fatal("old reader lost groups",err)}
 insights:=NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(base,"traffic.json")),nil);insights.now=func()time.Time{return now};usage,err:=insights.Get(context.Background(),id,"today");if err!=nil||usage.Result.UploadBytes!=1250||usage.Result.DownloadBytes!=3750{t.Fatal("old reader lost usage",err,usage)}
}
'''
source += r'''
func TestMultiLANLegacyFirewallRollback(t *testing.T){
 base:=os.Getenv("QUICKSTART_MULTI_LAN_UPGRADE_BASE");if base==""{t.Skip("explicit rollback fixture directory required")}
 if !filepath.IsAbs(base)||!strings.Contains(base,"/multi-lan/M10/legacy-data"){t.Fatal("unsafe rollback fixture")}
 dir:=filepath.Join(base,"config");const mac="02:00:00:00:00:55"
 if os.Getenv("QUICKSTART_MULTI_LAN_UPGRADE_PHASE")=="write"{
  if err:=os.MkdirAll(dir,0700);err!=nil{t.Fatal(err)}
  if err:=os.WriteFile(filepath.Join(dir,"firewall"),[]byte("config zone 'local'\n option name 'lan'\nconfig zone 'outside'\n option name 'wan'\n"),0600);err!=nil{t.Fatal(err)}
  if err:=writeDeviceAccessPolicyAt(dir,mac,false);err!=nil{t.Fatal(err)}
 }
 allowed,err:=readDeviceAccessPolicyAt(dir,mac);if err!=nil||allowed{t.Fatal("legacy reader lost paused access after backup restoration",err)}
}
'''
file=backend/'service/multi_lan_legacy_baseline_test.go';file.write_text(source)
(work/'fixture.json').write_text(json.dumps({'commit':'66589b22f140615db97851b6c3b1eec7aad975d5','fixtureOnly':True,'fixtureSha256':hashlib.sha256(source.encode()).hexdigest()},indent=2)+'\n')
print(backend)
