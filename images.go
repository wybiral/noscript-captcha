package main

import (
	"fmt"
	"math/rand"
)

// Return a random image path
func randImg() string {
	var x string
	if rand.Intn(2) == 0 {
		x = "c"
	} else {
		x = "d"
	}
	return fmt.Sprintf("img/%s%d.jpg", x, rand.Intn(1000))
}
