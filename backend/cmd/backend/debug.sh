
rm -f ./quickstart.x86_64
#CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build   -a -ldflags '-X main.BuildVersion='$GV' -X main.BuildDate='$DV' -s -w -extldflags "-static"' -o quickstart.arm64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build   -a -ldflags '-X main.BuildVersion='$GV' -X main.BuildDate='$DV' -s -w -extldflags "-static"' -o quickstart.x86_64
#CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build   -a -ldflags '-X main.BuildVersion='$GV' -X main.BuildDate='$DV' -s -w -extldflags "-static"' 
echo "build done"
#scp  ./backend  root@192.168.10.114:/tmp/backend
#scp  ./build/quickstart.arm64  root@192.168.9.118:/tmp/quickstart
scp  ./quickstart.x86_64  root@192.168.9.75:/tmp/quickstart
