package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("Starting pitank client")

	server := flag.String("server", "stream.pitank.com", "server host:port")
	name := flag.String("name", "pitank", "pitank name to use on registration")
	cameraID := flag.Int("camera", 0, "number of camera device to use")
	disableGPIO := flag.Bool("disable_gpio", false, "disables GPIO initialization")
	flag.Parse()

	camera := NewCamera(*cameraID)
	/*
		go camera.Process()
		for data := range camera.Stream {
			fmt.Println("Received data:", len(data))
		}
	*/
	//fmt.Println("Camera to use:", *cameraID)

	var commandProcessor CommandProcessor
	if *disableGPIO {
		commandProcessor = &FakePiTank{}
	} else {
		pitank, err := NewPiTank()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer pitank.Close()

		commandProcessor = pitank
	}

	// If websocket fails try to reopen it forever
	for {
		err := openWebsocket(*server, *name, commandProcessor, camera)
		fmt.Println("Retrying to connect to Websocket:", err)
		time.Sleep(5 * time.Second)
	}
}
