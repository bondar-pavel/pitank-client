package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc"
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
	Offer    string `json:"offer,ommitempty"`
	Answer   string `json:"answer,ommitempty"`
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

	peerConnection, err := initPeerConnection()
	if err != nil {
		fmt.Println("Peer connection init failed:", err)
	} else {
		offer, err := generateOffer(peerConnection)

		fmt.Println("Generated offer:", offer)
		if err != nil {
			fmt.Println("Offer generate failed:", err)
		} else {
			cmd := Command{Offer: offer}
			err := c.WriteJSON(cmd)
			if err != nil {
				fmt.Println("Error on writing offer:", err)
			}
		}
	}

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			fmt.Println("read:", err)
			return err
		}
		fmt.Printf("Received: %s", message)
		// Write message back to be able to measure round-trip delay
		err = c.WriteMessage(mt, message)
		if err != nil {
			fmt.Println("write:", err)
			return err
		}

		var cmd Command
		err = json.Unmarshal(message, &cmd)
		if err != nil {
			fmt.Println("Error on unmarshal:", err.Error())
			continue
		}

		if cmd.Answer != "" && peerConnection != nil {
			err := setWebRTCAnswer(peerConnection, cmd.Answer)
			if err != nil {
				fmt.Println("Error on receiving answer:", err)
			}
			continue
		}

		processCommand(cmd)
	}
}

// initializes PeerConnection with handlers for OnOpen, OnMessage callbacks
func initPeerConnection() (*webrtc.PeerConnection, error) {
	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// Create a datachannel with label 'data'
	dataChannel, err := peerConnection.CreateDataChannel("commands", nil)
	if err != nil {
		return nil, err
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Register channel opening handling
	dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())
		sendErr := dataChannel.SendText("Opened WebRTC connection")
		if sendErr != nil {
			fmt.Println("Can not send to channel:", sendErr)
		}
	})

	// Register text message handling
	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))

		var cmd Command
		err := json.Unmarshal(msg.Data, &cmd)
		if err != nil {
			fmt.Println("Error on unmarchal cmd:", string(msg.Data), err)
			return
		}
		processCommand(cmd)
	})

	return peerConnection, nil
}

// generateOffer generates WebRTC offer for peerConnection and encodes it,
// this offer should be passed to another WebRTC client to establish connection
func generateOffer(peerConnection *webrtc.PeerConnection) (string, error) {
	// Create an offer to send to the browser
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return "", err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		return "", err
	}

	return Encode(offer)
}

func setWebRTCAnswer(peerConnection *webrtc.PeerConnection, encodedAnswer string) error {
	// Wait for the answer to be pasted
	answer := webrtc.SessionDescription{}
	err := Decode(encodedAnswer, &answer)
	if err != nil {
		return err
	}

	// Apply the answer as the remote description
	err = peerConnection.SetRemoteDescription(answer)
	if err != nil {
		return err
	}
	return nil
}

// Encode encodes the input in base64
func Encode(obj interface{}) (string, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

// Decode decodes the input from base64
func Decode(in string, obj interface{}) error {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, obj)
	if err != nil {
		return err
	}
	return nil
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

	// If websocket fails try to reopen it forever
	for {
		err := openWebsocket(*server, *name)
		fmt.Println("Retrying to connect to Websocket:", err)
		time.Sleep(5 * time.Second)
	}
}
