package main

import (
	"bufio"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/experimental/devices/ccs811"
	"periph.io/x/periph/host"

	_ "github.com/lib/pq"
)

const (
	ccs811WaitAfterReset     = 2000 * time.Microsecond // The CCS811 needs a wait after reset
	ccs811WaitAfterAppStart  = 1000 * time.Microsecond // The CCS811 needs a wait after app start
	ccs811WaitAfterWake      = 50 * time.Microsecond   // The CCS811 needs a wait after WAKE signal
	ccs811WaitAfterAppErase  = 500 * time.Millisecond  // The CCS811 needs a wait after app erase (300ms from spec not enough)
	ccs811WaitAfterAppVerify = 70 * time.Millisecond   // The CCS811 needs a wait after app verify
	ccs811WaitAfterAppData   = 50 * time.Millisecond   // The CCS811 needs a wait after writing app data

	ccs811RegisterStatus        = 0x00
	ccs811RegisterMeasMode      = 0x01
	ccs811RegisterAlgResultData = 0x02 // up to 8 bytes
	ccs811RegisterRawData       = 0x03 // 2 bytes
	ccs811RegisterEnvData       = 0x05 // 4 bytes
	ccs811RegisterThresholds    = 0x10 // 5 bytes
	ccs811RegisterBaseline      = 0x11 // 2 bytes
	ccs811RegisterHWID          = 0x20
	ccs811RegisterHWVersion     = 0x21
	ccs811RegisterFWBootVersion = 0x23 // 2 bytes
	ccs811RegisterFWAppVersion  = 0x24 // 2 bytes
	ccs811RegisterErrorID       = 0xE0
	ccs811RegisterAppErase      = 0xF1 // 4 bytes
	ccs811RegisterAppData       = 0xF2 // 9 bytes
	ccs811RegisterAppVerify     = 0xF3 // 0 bytes
	ccs811RegisterAppStart      = 0xF4 // 0 bytes
	ccs811RegisterSWReset       = 0xFF // 4 bytes

	baselineFile = "BASELINE"
)

//var baseline = []byte{253, 184}

func main() {
	mode := os.Args[1]

	switch mode {
	case "measure":
		var conn = os.Args[2]
		measure(conn)

	case "flash":
		var imagePath = os.Args[2]
		flash(imagePath)

	case "test":
		test()

	default:
		fmt.Println("Please call climate monitor using one of these:")
		fmt.Println("climate-monitor measure <postgres connection string> - to output data to postgres")
		fmt.Println("climate-monitor flash <path to app image> - to flash ccs811 with new app firmware")
		fmt.Println("climate-monitor test")
		fmt.Println("")
	}
}

func test() {
	fmt.Println("I2C: Host init")
	_, err := host.Init()
	checkErr(err)

	fmt.Println("I2C: Open connection")
	bus, err := i2creg.Open("")
	checkErr(err)

	dev := i2c.Dev{Bus: bus, Addr: 0x5a}

	data := i2cRead(&dev, ccs811RegisterFWAppVersion, 2)
	fmt.Printf("%b\n", data)

	readDeviceStatus(&dev)

	i2cWrite(&dev, ccs811RegisterAppStart, []byte{})
	time.Sleep(ccs811WaitAfterAppStart)

	readDeviceStatus(&dev)
}

func measure(conn string) {
	var myIP = getOutboundIP()

	db, err := sql.Open("postgres", conn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected to DB!")

	fmt.Println("I2C: Host init")
	_, err = host.Init()
	checkErr(err)

	fmt.Println("I2C: Open connection")
	bus, err := i2creg.Open("")
	checkErr(err)

	fmt.Println("I2C: Get driver")
	dev, err := ccs811.New(bus, &ccs811.DefaultOpts)
	checkErr(err)

	mode, err := dev.GetMeasurementModeRegister()
	checkErr(err)

	fwData, err := dev.GetFirmwareData()
	checkErr(err)

	fmt.Println("")
	fmt.Println("========================================================================")
	fmt.Println("Device Information:")
	fmt.Println("========================================================================")
	fmt.Println("HW Model:     ", dev.String())
	fmt.Printf("HW Identifier: 0x%X\n", fwData.HWIdentifier)
	fmt.Printf("HW Version:    0x%X\n", fwData.HWVersion)
	fmt.Println("Boot Version: ", fwData.BootVersion)
	fmt.Println("App Version:  ", fwData.ApplicationVersion)
	fmt.Print("Mode:          ")

	switch mode.MeasurementMode {
	case ccs811.MeasurementModeIdle:
		fmt.Println("Idle, low power mode")
	case ccs811.MeasurementModeConstant1000:
		fmt.Println("Constant power mode, IAQ measurement every second")
	case ccs811.MeasurementModePulse:
		fmt.Println("Pulse heating mode IAQ measurement every 10 seconds")
	case ccs811.MeasurementModeLowPower:
		fmt.Println("Low power pulse heating mode IAQ measurement every 60 seconds")
	case ccs811.MeasurementModeConstant250:
		fmt.Println("Constant power mode, sensor measurement every 250ms")
	default:
		fmt.Println("Unknown")
	}

	fmt.Println("========================================================================")
	fmt.Println("Sensor Values:")
	fmt.Println("========================================================================")

	count := 0
	lowestBaseLine := loadBaseline()
	lowestBaseLineConverted := binary.LittleEndian.Uint16(lowestBaseLine)

	err = dev.SetBaseline(lowestBaseLine)
	checkErr(err)

	var val = ccs811.SensorValues{}
	err = dev.Sense(&val)
	checkErr(err)

	err = dev.SetBaseline(lowestBaseLine)
	checkErr(err)

	for true {
		status, err := dev.ReadStatus()

		if status&0x08 == 0x08 && err == nil {
			var val = ccs811.SensorValues{}
			err = dev.Sense(&val)
			checkErr(err)

			now := time.Now().UTC()

			baseline, err := dev.GetBaseline()
			checkErr(err)

			baselineConverted := binary.LittleEndian.Uint16(baseline)

			if baselineConverted < lowestBaseLineConverted {
				lowestBaseLine = baseline
				lowestBaseLineConverted = baselineConverted
				saveBaseline(baseline)
			}

			fmt.Print("Time: ", now.Format(time.RFC3339), ", ")
			fmt.Printf("Status: %b, ", status)
			fmt.Print("ECO2: ", val.ECO2, "ppm, ")
			fmt.Print("VOC: ", val.VOC, "ppb, ")
			fmt.Print("Current: ", val.RawDataCurrent, ", ")
			fmt.Print("Voltage: ", val.RawDataVoltage, ", ")
			fmt.Println("Baseline: ", baseline, baselineConverted)

			sqlStatement := `INSERT INTO ccs811 (time, ipaddress, baseline, eco2, etvoc, current, voltage) VALUES ($1, $2, $3, $4, $5, $6, $7)`
			_, err = db.Exec(sqlStatement, now, myIP.String(), baselineConverted, val.ECO2, val.VOC, val.RawDataCurrent, val.RawDataVoltage)
			checkErr(err)

			count = count + 1

			if count%300 == 0 {
				fmt.Println("setting baseline", lowestBaseLine, lowestBaseLineConverted)
				err = dev.SetBaseline(lowestBaseLine)
				checkErr(err)

				count = 0
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func loadBaseline() []byte {
	_, err := os.Stat(baselineFile)

	if os.IsNotExist(err) {
		return []byte{0xFF, 0xFF}
	}

	checkErr(err)

	file, err := os.Open(baselineFile)
	checkErr(err)
	defer file.Close()

	stats, err := file.Stat()
	checkErr(err)

	size := stats.Size()
	bytes := make([]byte, size)

	rdr := bufio.NewReader(file)
	_, err = rdr.Read(bytes)

	return bytes
}

func saveBaseline(baseline []byte) {
	file, err := os.Create(baselineFile)
	checkErr(err)
	defer file.Close()

	wrt := bufio.NewWriter(file)
	_, err = wrt.Write(baseline)
	checkErr(err)

	err = wrt.Flush()
	checkErr(err)
}

func flash(imagePath string) {
	imageBytes := loadFlashImage(imagePath)

	fmt.Println("I2C: Host init")
	_, err := host.Init()
	checkErr(err)

	fmt.Println("I2C: Open connection")
	bus, err := i2creg.Open("")
	checkErr(err)

	dev := i2c.Dev{Bus: bus, Addr: 0x5a}

	readDeviceStatus(&dev)
	swReset(&dev)
	readDeviceStatus(&dev)
	appErase(&dev)
	waitForAppErase(&dev)
	readDeviceStatus(&dev)
	writeApp(&dev, imageBytes)
	readDeviceStatus(&dev)
	appVerify(&dev)
	waitForAppVerify(&dev)
	readDeviceStatus(&dev)
	swReset(&dev)
	readDeviceStatus(&dev)
}

func waitForAppVerify(dev *i2c.Dev) {
	stopState := false

	for !stopState {
		status := i2cRead(dev, ccs811RegisterStatus, 1)[0]
		fmt.Printf("%x", status)
		stopState = status&0x20 == 0x20
		time.Sleep(ccs811WaitAfterAppVerify)
	}
}

func waitForAppErase(dev *i2c.Dev) {
	stopState := false

	for !stopState {
		status := i2cRead(dev, ccs811RegisterStatus, 1)[0]
		fmt.Printf("%x", status)
		stopState = status&0x40 == 0x40
		time.Sleep(ccs811WaitAfterAppErase)
	}
}

func writeApp(dev *i2c.Dev, bytes []byte) {
	size := len(bytes)
	count := 0

	for size > 0 {
		len := 8

		if size < 8 {
			len = size
		}

		//fmt.Printf("%x\n", bytes[count:count+len])

		i2cWrite(dev, ccs811RegisterAppData, bytes[count:count+len])

		count += len
		size -= len

		time.Sleep(ccs811WaitAfterAppData)

		if count%512 == 0 {
			fmt.Println(" ", size)
		}
	}

	if count%512 != 0 {
		fmt.Println(" ", size)
	}
}

func appVerify(dev *i2c.Dev) {
	fmt.Println("I2C: Call APP_VERIFY register")
	i2cWrite(dev, ccs811RegisterAppVerify, []byte{})
	time.Sleep(ccs811WaitAfterAppVerify)
}

func appErase(dev *i2c.Dev) {
	fmt.Println("I2C: Call APP_ERASE register")
	i2cWrite(dev, ccs811RegisterAppErase, []byte{0xe7, 0xa7, 0xe6, 0x09})
	time.Sleep(ccs811WaitAfterAppErase)
}

func swReset(dev *i2c.Dev) {
	fmt.Println("I2C: Call SW_RESET register")
	i2cWrite(dev, ccs811RegisterSWReset, []byte{0x11, 0xe5, 0x72, 0x8a})
	time.Sleep(ccs811WaitAfterReset)
}

func i2cWrite(dev *i2c.Dev, register byte, bytesToWrite []byte) {
	data := append([]byte{register}, bytesToWrite...)
	err := dev.Tx(data, nil)
	checkErr(err)
}

func i2cRead(dev *i2c.Dev, register byte, outBytes int) []byte {
	data := make([]byte, outBytes)
	err := dev.Tx([]byte{register}, data)
	checkErr(err)
	return data
}

func readDeviceStatus(dev *i2c.Dev) {
	data := i2cRead(dev, ccs811RegisterStatus, 1)
	fmt.Printf("I2C: Device status %08b\n", data[0])
}

func loadFlashImage(imagePath string) []byte {
	fmt.Println("Loading flash image:", imagePath)
	file, err := os.Open(imagePath)
	checkErr(err)
	defer file.Close()

	stats, err := file.Stat()
	checkErr(err)

	size := stats.Size()
	fmt.Println("Bytes:", size)

	bytes := make([]byte, size)

	rdr := bufio.NewReader(file)
	_, err = rdr.Read(bytes)

	return bytes
}

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
