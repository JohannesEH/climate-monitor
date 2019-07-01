package main

import (
	"fmt"
	"time"

	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/experimental/devices/ccs811"
	"periph.io/x/periph/host"
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	host.Init()

	bus, err := i2creg.Open("")
	checkErr(err)

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

	for ; true; {
		status, err := dev.ReadStatus()
		checkErr(err)

		if status & 0x08 == 0x08 {
			var val = ccs811.SensorValues{}
			err = dev.Sense(&val)
			checkErr(err)
			baseline, err := dev.GetBaseline()
			checkErr(err)
			fmt.Printf("Status: %b, ", status)
			fmt.Print("Baseline: ", baseline, ", ")
			fmt.Print("ECO2: ", val.ECO2, "ppm, ")
			fmt.Print("VOC: ", val.VOC, "ppb, ")
			fmt.Print("Current: ", val.RawDataCurrent, ", ")
			fmt.Println("Voltage: ", val.RawDataVoltage)
		}

		time.Sleep(50 * time.Millisecond)
	}


	// fmt.Println("Err:          ", val.Error)
}
