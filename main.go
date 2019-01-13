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

var CommandToGPIO = map[string]rpio.Pin{
	"trackleft_forward":  rpio.Pin(7),  // PIN: 26 GPIO: 7
	"trackleft_reverse":  rpio.Pin(8),  // PIN: 24 GPIO: 8
	"trackright_forward": rpio.Pin(11), // PIN: 23 GPIO: 11
	"trackright_reverse": rpio.Pin(25), // PIN: 22 GPIO: 25
	"tower_left":         rpio.Pin(9),  // PIN: 21 GPIO: 9
	"tower_right":        rpio.Pin(10), // PIN: 19 GPIO: 10
}

// DisallowedCombinations maps disallowed pairs
// of simultanious commands to protect from short-circute
// on some types of hardware
var DisallowedCombinations = map[string]string{
	"trackleft_forward":  "trackleft_reverse",
	"trackleft_reverse":  "trackleft_forward",
	"trackright_forward": "trackright_reverse",
	"trackright_reverse": "trackright_forward",
	"tower_left":         "tower_right",
	"tower_right":        "tower_left",
}

type Command struct {
	Commands string `json:"commands"`
}

// initializePins sets GPIO pins as outputs with low state
func initializePins() {
	for _, gpio := range CommandToGPIO {
		gpio.Output()
		gpio.Low()
	}
}

// resetPins set all pins to low state
func resetPins() {
	for _, gpio := range CommandToGPIO {
		gpio.Low()
	}
}

// setPins sets pin state according to state map
func setPins(stateMap map[string]bool) {
	b, err := json.Marshal(stateMap)
	fmt.Println("Setting outputs:", string(b), err)

	for cmd, state := range stateMap {
		gpio, exist := CommandToGPIO[cmd]
		if !exist {
			if cmd == "stop" {
				fmt.Println("Stopping...")
			} else {
				fmt.Println("Command not found:", cmd)
			}
			continue
		}

		if state {
			gpio.High()
		} else {
			gpio.Low()
		}
	}
}

// getStateMap receives list of valid commands and
// produces map with allowed combination of hi/low pin states
func getStateMap(commands []string) map[string]bool {
	// initialize fresh state map
	stateMap := make(map[string]bool)
	for key := range CommandToGPIO {
		stateMap[key] = false
	}

	for _, cmd := range commands {
		// set command to stateMap and check for conflicting states
		disallowed, exist := DisallowedCombinations[cmd]
		if exist {
			// if we tring to set disallowed state, cleanup both conflicting values
			if stateMap[disallowed] {
				fmt.Printf("%s and %s are conflicting commands, cleaning up both\n", cmd, disallowed)
				stateMap[disallowed] = false
				continue
			}
		}
		// cmd is allowed, so set it to state map
		stateMap[cmd] = true
	}
	return stateMap
}

// processCommand parses command content and set allowed combination of GPIO pins
func processCommand(c Command) {
	if c.Commands == "" {
		fmt.Println("No command to run, skipping")
		return
	}

	validCommands := make([]string, 0)
	cmds := strings.Split(c.Commands, ",")
	for _, cmd := range cmds {
		_, exist := CommandToGPIO[cmd]
		if !exist {
			fmt.Println("Unknown command:", cmd)
			continue
		}
		validCommands = append(validCommands, cmd)
	}

	if len(validCommands) == 0 {
		resetPins()
		return
	}

	stateMap := getStateMap(validCommands)
	// set stateMap as GPIO output state
	setPins(stateMap)
}

func openWebsocket(host, name string) error {
	u := url.URL{Scheme: "ws", Host: host, Path: "/api/connect/" + name}
	fmt.Printf("connecting to %s\n", u.String())

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

	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Unmap gpio memory when done
	defer rpio.Close()

	initializePins()

	openWebsocket(*server, *name)
}
