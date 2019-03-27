package gotfp

import "time"

func resetTimer(timer *time.Timer, duration time.Duration) {
	timer.Stop()
	select {
	case <-timer.C: //Try to drain from the channel.
	default:
	}
	timer.Reset(duration)
}
