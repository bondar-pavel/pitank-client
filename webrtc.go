package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/pion/webrtc/v2"
)

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

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	peerConnection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
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
			// Write message back to be able to measure round-trip delay
			err = dataChannel.SendText(string(msg.Data))
			if err != nil {
				fmt.Println("Error on writing back:", err)
			}
			fmt.Println("Writing back:", string(msg.Data))

			processCommand(cmd)
		})
	})

	return peerConnection, nil
}

func setWebRTCOffer(peerConnection *webrtc.PeerConnection, encodedOffer string) (string, error) {
	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	err := Decode(encodedOffer, &offer)
	if err != nil {
		return "", err
	}

	// Apply the remote offer as the remote description
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		return "", err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		return "", err
	}

	return Encode(answer)
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
