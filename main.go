package main

import (
	"fmt"
	"os"

	rpio "github.com/stianeikeland/go-rpio"
)

// map PIN number on the board into GPIO number on bcm2835
var pinToGPIO = map[int]int{
	11: 17,
	12: 18, // PIN 12: GPIO18
	16: 23,
	18: 24,
	19: 10, // pins 19-26 are used for hardware v2
	21: 9,
	22: 25,
	23: 11,
	24: 8,
	26: 7,
}

var commandToPin = map[string]int{
	"left_forward":   26,
	"left_backward":  24,
	"right_forward":  23,
	"right_backward": 22,
	"tower_left":     21,
	"tower_right":    19,
}

func main() {
	fmt.Println("Starting pitank client")

	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmap gpio memory when done
	defer rpio.Close()
}
