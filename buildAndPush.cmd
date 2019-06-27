set GOOS=linux
set GOARCH=arm
set GOARM=7

go install -v && go build && scp ./climate-monitor pi@192.168.2.215:~