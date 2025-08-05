//go:build benchmark
// +build benchmark

package scheduler

import (
	"math/rand"
	"testing"
	"time"

	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/scheduler/defscheduler"
	"github.com/lonng/nano/scheduler/twscheduler"
)

const taskNum = 100000

func TestDefault_Stress(t *testing.T) {
	scheduler := defscheduler.NewScheduler("test", 500*time.Millisecond)
	scheduler.Start()
	defer scheduler.Close()

	for i := 0; i < taskNum; i++ {
		idx := i
		go func() {
			delay := rand.Int63n(30 * int64(time.Second))
			time.Sleep(time.Duration(delay))
			scheduler.NewTimer(30*time.Second, func() {
				log.Info("timer %v", idx)
			})
		}()
	}

	time.Sleep(2 * time.Minute)
}

func TestTimeWheel_Stress(t *testing.T) {
	scheduler := twscheduler.NewScheduler("test", 2500*time.Millisecond, 16)
	scheduler.Start()
	defer scheduler.Close()

	for i := 0; i < taskNum; i++ {
		idx := i
		go func() {
			delay := rand.Int63n(30 * int64(time.Second))
			time.Sleep(time.Duration(delay))
			scheduler.NewTimer(30*time.Second, func() {
				log.Info("timer %v", idx)
			})
		}()
	}

	time.Sleep(2 * time.Minute)
}
