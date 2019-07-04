set GOOS=linux
set GOARCH=arm
set GOARM=7

go install -v && go build && scp ./climate-monitor pi@192.168.2.215:~

rem && scp ./CCS811_FW_App_v2-0-0.bin pi@192.168.2.215:~ && scp ./CCS811_FW_App_v2-0-1.bin pi@192.168.2.215:~