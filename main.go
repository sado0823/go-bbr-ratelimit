package main

import (
	"go-bbr-ratelimit/internal/middleware"
	"math"
	"math/rand"
	"time"
)

func main() {

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for i := 0; i < 100; i++ {
		i := i
		go func() {
			for {
				var b float64
				b = math.Sqrt(float64(i)) * rand.Float64()
				_ = b
			}
		}()
	}

	for range ticker.C {
		middleware.CpuProc()
	}

	for {
	}
}
