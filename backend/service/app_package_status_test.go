package service
import("errors";"os";"path/filepath";"testing")
func TestPackageStatusFailsClosedOnUnreadableDatabase(t *testing.T){
 file:=filepath.Join(t.TempDir(),"control");os.WriteFile(file,nil,0600);info,_:=os.Stat(file)
 for _,failure:=range []error{os.ErrPermission,errors.New("I/O error")} {
  installed,err:=packageInstalled("app-meta-fastnet",func(path string)(os.FileInfo,error){if path=="/etc/apk/repositories.d/distfeeds.list"{return nil,os.ErrNotExist};return nil,failure})
  if installed||err==nil{t.Fatalf("unreadable package database became absent: %v %v",installed,err)}
 }
 installed,err:=packageInstalled("app-meta-fastnet",func(path string)(os.FileInfo,error){if path=="/etc/apk/repositories.d/distfeeds.list"{return nil,os.ErrNotExist};return info,nil})
 if !installed||err!=nil{t.Fatal(installed,err)}
 installed,err=packageInstalled("app-meta-fastnet",func(string)(os.FileInfo,error){return nil,os.ErrNotExist});if installed||err!=nil{t.Fatal(installed,err)}
}
