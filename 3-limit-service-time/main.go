//////////////////////////////////////////////////////////////////////
//
// Your video processing service has a freemium model. Everyone has 10
// sec of free processing time on your service. After that, the
// service will kill your process, unless you are a paid premium user.
//
// Beginner Level: 10s max per request
// Advanced Level: 10s max per user (accumulated)
//

package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

// User defines the UserModel. Use this to check whether a User is a
// Premium user or not
type User struct {
	ID        int
	IsPremium bool
	TimeUsed  atomic.Uint64 // in seconds
}

// HandleRequest runs the processes requested by users. Returns false
// if process had to be killed
func HandleRequest(process func(), u *User) bool {
	if !u.IsPremium && u.TimeUsed.Load() >= 10 {
		fmt.Printf("user %d time limit execeeded", u.ID)
		return false
	}

	done := make(chan bool)
	go func() {
		process()
		done <- true
	}()

	if u.IsPremium {
		<-done
		return true
	}

	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-done:
			return true
		case <-t.C:
			if used := u.TimeUsed.Add(1); used >= 10 {
				fmt.Printf("user %d used %d seconds\n", u.ID, used)
				return false
			}
		}
	}
}

func main() {
	RunMockServer()
}
