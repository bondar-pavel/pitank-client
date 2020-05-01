// +build camera

package main

import (
	"fmt"

	"gocv.io/x/gocv"
)

// Camera reads stream from webcam and publish each frame
// to stream channel as []byte
type Camera struct {
	CameraID  int
	Stream    chan []byte
	cancel    chan bool
	isStarted bool
}

// NewCamera create new instance of Camera
func NewCamera(cameraID int) *Camera {
	return &Camera{
		CameraID: cameraID,
		Stream:   make(chan []byte, 1),
		cancel:   make(chan bool, 1),
	}
}

// Start starts video stream if not started already
func (c *Camera) Start() {
	fmt.Println("Starting camera")
	if !c.isStarted {
		go c.Process()
		c.isStarted = true
	}
}

// Stop initiate stop of camera stream is stream was previously started
func (c *Camera) Stop() {
	if c.isStarted {
		c.cancel <- true
		c.isStarted = false
	}
}

// Process opens webcam, read each frame and publish to stream channel
func (c *Camera) Process() error {
	fmt.Println("Opening camera device:", c.CameraID)
	webcam, err := gocv.OpenVideoCapture(c.CameraID)
	if err != nil {
		return err
	}
	defer webcam.Close()
	img := gocv.NewMat()

	for {
		select {
		case <-c.cancel:
			close(c.Stream)
			// recreate stream in case we want to start it again
			c.Stream = make(chan []byte, 1)
			return nil
		default:
		}

		fmt.Println("Getting new frame")
		if ok := webcam.Read(&img); !ok {
			continue
		}

		fmt.Println("Encoding frame to jpg")
		buf, err := gocv.IMEncode(".jpg", img)
		if err != nil {
			continue
		}
		c.Stream <- buf
	}
}
