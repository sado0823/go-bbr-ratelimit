package middleware

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewBBR(t *testing.T) {

	bbr := NewBBR(
		WithWindow(time.Second*5),
		WithBuckets(50),
		WithCpuThreshold(100),
	)

	var (
		wg   sync.WaitGroup
		drop int64
	)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 300; i++ {
				f, err := bbr.Allow()
				if err != nil {
					atomic.AddInt64(&drop, 1)
				} else {
					count := rand.Intn(100)
					time.Sleep(time.Millisecond * time.Duration(count))
					f.Pass()
				}
			}
		}()
	}
	wg.Wait()
	fmt.Println("drop: ", drop)

}
