package main

import (
	"gocv.io/x/gocv"
)

// Camera reads stream from webcam and publish each frame
// to stream channel as []byte
type Camera struct {
	CameraID int
	Stream   chan []byte
	cancel   chan bool
}

// NewCamera create new instance of Camera
func NewCamera(cameraID int) *Camera {
	return &Camera{
		CameraID: cameraID,
		Stream:   make(chan []byte, 1),
		cancel:   make(chan bool, 1),
	}
}

// Stop initiate stop of camera stream
func (c Camera) Stop() {
	c.cancel <- true
}

// Process opens webcam, read each frame and publish to stream channel
func (c Camera) Process() error {
	webcam, err := gocv.OpenVideoCapture(c.CameraID)
	if err != nil {
		return err
	}
	defer webcam.Close()
	img := gocv.NewMat()

	for {
		select {
		case <-c.cancel:
			return nil
		default:
		}
		if ok := webcam.Read(&img); !ok {
			continue
		}

		buf, err := gocv.IMEncode(".jpg", img)
		if err != nil {
			continue
		}
		c.Stream <- buf
	}
}
