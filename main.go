package main

import (
	"fmt"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

func main() {
	err := rpio.Open()
	if err != nil {
		panic(err)
	}
	defer rpio.Close()

	pin := rpio.Pin(17)

	pin.Output()
	pin.Low()
	time.Sleep(time.Microsecond * 1000)
	pin.Input()
	pin.PullUp()

	pin.Detect(rpio.FallEdge)

	for {
		if pin.EdgeDetected() {
			fmt.Printf("Edge detected! State: %d\n", pin.Read())
		}
		time.Sleep(time.Microsecond * 5)
	}
}
