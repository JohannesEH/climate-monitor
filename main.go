package main

import (
	"fmt"

	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/experimental/devices/ccs811"
	"periph.io/x/periph/host"
)

func main() {
	host.Init()

	bus, err := i2creg.Open("")
	if err != nil {
		panic(err)
	}

	dev, err := ccs811.New(bus, &ccs811.DefaultOpts)
	if err != nil {
		panic(err)
	}

	status, err := dev.ReadStatus()
	if err != nil {
		panic(err)
	}

	mode, err := dev.GetMeasurementModeRegister()
	if err != nil {
		panic(err)
	}

	fwData, err := dev.GetFirmwareData()
	if err != nil {
		panic(err)
	}

	baseline, err := dev.GetBaseline()
	if err != nil {
		panic(err)
	}

	var val = ccs811.SensorValues{}
	err = dev.Sense(&val)
	if err != nil {
		panic(err)
	}

	fmt.Println("")
	fmt.Println("========================================================================")
	fmt.Println("Device Information:")
	fmt.Println("========================================================================")
	fmt.Println("HW Model:     ", dev.String())
	fmt.Printf("HW Identifier: 0x%X\n", fwData.HWIdentifier)
	fmt.Printf("HW Version:    0x%X\n", fwData.HWVersion)
	fmt.Println("Current:      ", val.RawDataCurrent)
	fmt.Println("Voltage:      ", val.RawDataVoltage)
	fmt.Println("Boot Version: ", fwData.BootVersion)
	fmt.Println("App Version:  ", fwData.ApplicationVersion)
	fmt.Printf("Status:        %b\n", status)
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

	fmt.Println("Baseline:     ", baseline)

	fmt.Println("")
	fmt.Println("")
	fmt.Println("========================================================================")
	fmt.Println("Sensor Values:")
	fmt.Println("========================================================================")
	fmt.Println("ECO2:         ", val.ECO2, "ppm")
	fmt.Println("VOC:          ", val.VOC, "ppb")
	fmt.Println("")
	fmt.Println("")
	// fmt.Println("Err:          ", val.Error)
}
