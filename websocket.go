package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func openWebsocket(host, name string) error {
	u := url.URL{Scheme: "ws", Host: host, Path: "/api/connect/" + name}
	fmt.Printf("connecting to %s\n", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return errors.Wrap(err, "Error on Dial")
	}
	defer c.Close()

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

		if cmd.Offer != "" {
			peerConnection, err := initPeerConnection()
			if err != nil {
				fmt.Println("Peer connection init failed:", err)
				continue
			}
			answer, err := setWebRTCOffer(peerConnection, cmd.Offer)
			if err != nil {
				fmt.Println("Error on receiving answer:", err)
				continue
			}
			fmt.Println("Writing answer:", answer)
			reply := Command{Answer: answer}
			c.WriteJSON(reply)
			continue
		}

		processCommand(cmd)
	}
}
