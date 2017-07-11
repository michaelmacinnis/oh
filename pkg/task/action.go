// Released under an MIT license. See LICENSE.

package task

import "sync"

const (
	running = iota
	suspended
	terminated
)

type action struct {
	*sync.Cond
	state int
}

func NewAction() *action {
	return &action{sync.NewCond(&sync.RWMutex{}), running}
}

func (a *action) Continue() {
	//println("continue")
	a.L.Lock()
	defer a.L.Unlock()

	if a.state != suspended {
		panic("can't continue a task that has not been suspended")
	}

	a.state = running
	a.Signal()
}

func (a *action) Runnable() bool {
	//println("runnable")
	a.L.Lock()
	defer a.L.Unlock()

	for a.state == suspended {
		a.Wait()
	}

	return a.state == running
}

func (a *action) Suspend() {
	//println("suspend")
	a.L.Lock()
	defer a.L.Unlock()

	if a.state != running {
		panic("can't suspend a task that is not running")
	}

	a.state = suspended
}

func (a *action) Terminate() {
	//println("terminate")
	a.L.Lock()
	defer a.L.Unlock()

	previous := a.state
	if previous == terminated {
		panic("can't terminate a task that has already been terminated")
	}

	a.state = terminated

	if previous == suspended {
		a.Signal()
	}
}
