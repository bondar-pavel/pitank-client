package main

import (
	"encoding/json"
	"fmt"
	"strings"

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
	Offer    string `json:"offer,ommitempty"`
	Answer   string `json:"answer,ommitempty"`
	Camera   string `json:"camera,ommitempty"`
	Time     int64  `json:"time,omitempty"`
}

type CommandProcessor interface {
	ProcessCommand(Command)
}

type PiTank struct {
	commandToGPIO          map[string]rpio.Pin
	disallowedCombinations map[string]string
}

func NewPiTank() (*PiTank, error) {
	// Open and map memory to access gpio, check for errors
	if err := rpio.Open(); err != nil {
		return nil, err
	}

	p := &PiTank{
		commandToGPIO:          CommandToGPIO,
		disallowedCombinations: DisallowedCombinations,
	}
	p.initializePins()
	return p, nil
}

// Close unmaps gpio memory when done
func (p PiTank) Close() {
	rpio.Close()
}

// initializePins sets GPIO pins as outputs with low state
func (p PiTank) initializePins() {
	for _, gpio := range p.commandToGPIO {
		gpio.Output()
		gpio.Low()
	}
}

// resetPins set all pins to low state
func (p PiTank) resetPins() {
	for _, gpio := range p.commandToGPIO {
		gpio.Low()
	}
}

// setPins sets pin state according to state map
func (p PiTank) setPins(stateMap map[string]bool) {
	b, err := json.Marshal(stateMap)
	fmt.Println("Setting outputs:", string(b), err)

	for cmd, state := range stateMap {
		gpio, exist := p.commandToGPIO[cmd]
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
func (p PiTank) getStateMap(commands []string) map[string]bool {
	// initialize fresh state map
	stateMap := make(map[string]bool)
	for key := range p.commandToGPIO {
		stateMap[key] = false
	}

	for _, cmd := range commands {
		// set command to stateMap and check for conflicting states
		disallowed, exist := p.disallowedCombinations[cmd]
		if exist {
			// if we trying to set disallowed state, cleanup both conflicting values
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

// ProcessCommand parses command content and set allowed combination of GPIO pins
func (p PiTank) ProcessCommand(c Command) {
	if c.Commands == "" {
		fmt.Println("No command to run, skipping")
		return
	}

	validCommands := make([]string, 0)
	cmds := strings.Split(c.Commands, ",")
	for _, cmd := range cmds {
		_, exist := p.commandToGPIO[cmd]
		if !exist {
			fmt.Println("Unknown command:", cmd)
			continue
		}
		validCommands = append(validCommands, cmd)
	}

	if len(validCommands) == 0 {
		p.resetPins()
		return
	}

	stateMap := p.getStateMap(validCommands)
	// set stateMap as GPIO output state
	p.setPins(stateMap)
}
