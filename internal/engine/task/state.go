// Released under an MIT license. See LICENSE.

package task

import (
	"sync"

	"github.com/michaelmacinnis/oh/internal/common/interface/cell"
)

// E R S W
// 0 1 0 X Task is running.
// 0 1 1 X Task is stopping but Runnable has not yet been called.
// 0 0 1 X Task is stopping but has not returned from Run.
// 0 0 0 X Task is stopped.

// E R S W
// X X X 0 Task is not waiting.
// X X X 1 Task is waiting for an external process to finish.

// E R S W
// 1 X X X Task is no longer runnable.

// The type state is a task's state.
type state struct {
	*sync.Mutex
	resume  *sync.Cond
	stopped *sync.Cond

	result cell.I

	exited   bool
	running  bool
	stopping bool
	waiting  bool
}

func fresh() *state {
	m := &sync.Mutex{}

	return &state{Mutex: m, resume: sync.NewCond(m), stopped: sync.NewCond(m)}
}

func (s *state) Exit() {
	s.Lock()
	defer s.Unlock()

	s.exited = true
}

func (s *state) Exited() bool {
	s.Lock()
	defer s.Unlock()

	return s.exited
}

// Notify notifies a waiting task that it can continue. If running is set,
// the condition variable resume is used to signal the task to resume.
// Calling this on a task that is not waiting results in a panic.
//
// R S W -> R S W
// X X 0    X X 0 panic
// 0 X 1    0 X 0
// 1 X 1    1 X 0 resume signal.
//
func (s *state) Notify(r cell.I) {
	s.Lock()
	defer s.Unlock()

	if !s.waiting {
		panic("can't resume a task that isn't waiting.")
	}

	s.result = r
	s.waiting = false

	if s.running {
		s.resume.Signal()
	}
}

// Runnable returns true if a task is running and not stopping or waiting.
// If the task is waiting and not stopping Runnable blocks until signaled via
// the resume condition variable.
//
// R S W -> R S W
// 0 X X    0 X X returns false
// 1 0 0    1 0 0 returns true
// 1 0 1    0 1 1 blocks until stopped, ...
// 1 0 1    1 0 0 ...or resumed
// 1 1 0    0 1 0 returns false
// 1 1 1    0 1 1 .
//
func (s *state) Runnable() bool {
	s.Lock()
	defer s.Unlock()

	if !s.running {
		return false
	}

	for s.waiting && !s.stopping {
		s.resume.Wait()
	}

	s.running = !s.stopping

	return s.running
}

// Started marks the task as running.
// Calling this on a task that is already running results in a panic.
//
// R S W -> R S W
// 0 0 X -> 1 0 X
// 0 1 X -> 0 0 X
// 1 X X    1 X X panic.
//
func (s *state) Started() {
	s.Lock()
	defer s.Unlock()

	if s.running {
		// Why is this already running?
		panic("already running")
	}

	s.running = true
}

// Stop marks a running task as stopping and waits for a signal that the task
// has stopped. If the task is waiting Stop signals it to resume so that it
// can unblock and signal that it has stopped.
//
// R S W -> R S W
// X 1 X -> X 1 X
// 0 X X -> 0 X X
// 1 0 0 -> 0 0 0 waits for stopped signal
// 1 0 1 -> 0 0 1 resumes tasks, waits for stopped signal.
//
func (s *state) Stop(f func()) {
	s.Lock()
	defer s.Unlock()

	if s.stopping {
		return
	}

	if !s.running {
		// Stopped running before we got here.
		return
	}

	s.stopping = true

	if f != nil {
		f()
	}

	if s.waiting {
		s.resume.Signal()
	}

	for !s.stopping {
		s.stopped.Wait()
	}
}

// Stopped clears running and stopping and if stopping was set, signals that
// the task has stopped.
//
// R S W -> R S W
// X 0 X -> 0 0 X
// 0 1 X -> 0 0 X signals task has stopped.
//
func (s *state) Stopped() {
	s.Lock()
	defer s.Unlock()

	signal := s.stopping

	s.running = false
	s.stopping = false

	if signal {
		s.stopped.Signal()
	}
}

// Value returns the most recent value set by notify.
func (s *state) Value() cell.I {
	s.Lock()
	defer s.Unlock()

	r := s.result
	s.result = nil

	return r
}

// Wait sets waiting. Calling this when waiting is already set panics.
//
// R S W -> R S W
// X X 0 -> X X 1
// X X 1 -> X X 1 panic.
//
func (s *state) Wait() {
	s.Lock()
	defer s.Unlock()

	if s.waiting {
		// Why is this already waiting?
		panic("already waiting")
	}

	s.waiting = true
}
