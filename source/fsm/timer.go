package fsm

import (
	"time"
)

func InitDoorTimer(finishedC chan<- bool, startC <-chan bool, doorOpenTime time.Duration) {

	doorTimer := time.NewTimer(doorOpenTime)

	/* Drain channel if expired */
	if !doorTimer.Stop() {
		<-doorTimer.C
	}

	for {
		select {
		case <-startC:
			doorTimer.Reset(doorOpenTime)
		case <-doorTimer.C:
			finishedC <- true
		}
	}
}
