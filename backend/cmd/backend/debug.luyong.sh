
# source /Users/luyong/work/Go/go1.20.sh
# CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build   -a -ldflags '-X main.BuildVersion='$GV' -X main.BuildDate='$DV' -s -w -extldflags "-static"' -o backend
# CGO_ENABLED=0  go build   -a -ldflags '-X main.BuildVersion='$GV' -X main.BuildDate='$DV' -s -w -extldflags "-static"' -o quickstart.arm64
# CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build   -a -ldflags '-X main.BuildVersion='$GV' -X main.BuildDate='$DV' -s -w -extldflags "-static"' -o quickstart.x86_64
# CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build   -a -ldflags '-X main.BuildVersion='$GV' -X main.BuildDate='$DV' -s -w -extldflags "-static"' 
echo "build done"
#  scp  ./backend  root@192.168.9.18:/tmp/backend

#  scp  quickstart.x86_64 root@192.168.9.242:/tmp/quickstart.x86_64
# scp  ./build/quickstart.arm64  root@192.168.9.114:/tmp/quickstart
# cp ./backend /Volumes/istorebackend/
# cp ./backend /Volumes/istorebackend114

#18
# rm ./backend
# CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build   -a -ldflags '-X main.BuildVersion='$GV' -X main.BuildDate='$DV' -s -w -extldflags "-static"' -o backend
# scp  ./backend  root@192.168.9.18:/tmp/backend

#242
rm ./quickstart.x86_64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build   -a -ldflags '-X main.BuildVersion='$GV' -X main.BuildDate='$DV' -s -w -extldflags "-static"' -o quickstart.x86_64
scp  quickstart.x86_64 root@192.168.9.242:/root/quickstart.x86_64
# scp  quickstart.x86_64 root@192.168.9.18:/mnt/data_md1