package main

import "fmt"
import "github.com/stianeikeland/go-rpio/v4"

func main() {
	fmt.Println("Hello World")
	rpio.Open()

	// pin := rpio.Pin(10)

	// pin.Output() // Output mode
	// pin.High()   // Set pin High
	// pin.Low()    // Set pin Low
	// pin.Toggle() // Toggle pin (Low -> High -> Low)

	// pin.Input()       // Input mode
	// res := pin.Read() // Read state from pin (High / Low)

	// pin.Mode(rpio.Output) // Alternative syntax
	// pin.Write(rpio.High)  // Alternative syntax

	// pin.PullUp()
	// pin.PullDown()
	// pin.PullOff()

	// pin.Pull(rpio.PullUp)

	rpio.Close()
	fmt.Println("Bye World")
}
