package main

import (
	"fmt"

	rpio "github.com/stianeikeland/go-rpio"
)

func main() {
	fmt.Println("Starting pitank client.")

	err := rpio.Open()
	fmt.Println(err)
}
