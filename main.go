package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
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

var CommandToGPIO = map[string]rpio.Pin{
	"left_forward":   rpio.Pin(7),  // PIN: 26 GPIO: 7
	"left_backward":  rpio.Pin(8),  // PIN: 24 GPIO: 8
	"right_forward":  rpio.Pin(11), // PIN: 23 GPIO: 11
	"right_backward": rpio.Pin(25), // PIN: 22 GPIO: 25
	"tower_left":     rpio.Pin(9),  // PIN: 21 GPIO: 9
	"tower_right":    rpio.Pin(10), // PIN: 19 GPIO: 10
}

type Command struct {
	Commands string `json:"commands"`
}

func resetPins() {
	for _, gpio := range CommandToGPIO {
		gpio.Low()
	}
}

func processCommand(c Command) {
	if c.Commands == "" {
		fmt.Println("No command to run, skipping")
		return
	}

	cmds := strings.Split(c.Commands, ",")
	for _, cmd := range cmds {
		gpio, exist := CommandToGPIO[cmd]
		if !exist {
			fmt.Println("Unknown command:", cmd)
			resetPins()
			continue
		}
		gpio.High()
	}
}

func openWebsocket(host, name string) error {
	u := url.URL{Scheme: "ws", Host: host, Path: "/api/connect/" + name}
	fmt.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return errors.Wrap(err, "Error on Dial")
	}
	defer c.Close()

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return err
		}
		fmt.Printf("Received: %s", message)

		var cmd Command
		err = json.Unmarshal(message, &cmd)
		if err != nil {
			fmt.Println("Error on unmarshal:", err.Error())
			continue
		}
		processCommand(cmd)
	}
}

func main() {
	fmt.Println("Starting pitank client")

	server := flag.String("server", "stream.pitank.com", "server host:port")
	name := flag.String("name", "pitank", "pitank name to use on registration")
	flag.Parse()

	openWebsocket(*server, *name)

	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	// Initialize GPIO pins as outputs with low state
	for _, gpio := range CommandToGPIO {
		gpio.Output()
		gpio.Low()
	}
}
