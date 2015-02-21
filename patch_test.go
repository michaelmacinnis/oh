package main

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"unsafe"
)

type handle struct {
	*exec.Cmd
	io.WriteCloser
	*testing.T
}

func (h *handle) Stop() {
	h.Close()
	h.Wait()
}

func child(cmd *handle) (pid, pgrp int) {
	pid = cmd.Process.Pid

	pgrp, err := syscall.Getpgid(pid)
	if err != nil {
		cmd.Fatal(err)
	}

	return
}

func command(t *testing.T) *handle {
	cmd := exec.Command("cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	return &handle{cmd, stdin, t}
}

func parent() (pid, pgrp int) {
	return syscall.Getpid(), syscall.Getpgrp()
}

func TestZeroSysProcAttr(t *testing.T) {
	ppid, ppgrp := parent()

	cmd := command(t)

	cmd.Start()
	defer cmd.Stop()

	cpid, cpgrp := child(cmd)

	if cpid == ppid {
		t.Fatalf("Parent and child have the same process ID")
	}

	if cpgrp != ppgrp {
		t.Fatalf("Child is not in parent's process group")
	}
}

func TestSetpgid(t *testing.T) {
	ppid, ppgrp := parent()

	cmd := command(t)

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Start()
	defer cmd.Stop()

	cpid, cpgrp := child(cmd)

	if cpid == ppid {
		t.Fatalf("Parent and child have the same process ID")
	}

	if cpgrp == ppgrp {
		t.Fatalf("Parent and child are in the same process group")
	}

	if cpid != cpgrp {
		t.Fatalf("Child's process group is not the child's process ID")
	}
}

func TestPgid(t *testing.T) {
	ppid, ppgrp := parent()

	cmd1 := command(t)

	cmd1.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd1.Start()
	defer cmd1.Stop()

	cpid1, cpgrp1 := child(cmd1)

	if cpid1 == ppid {
		t.Fatalf("Parent and child 1 have the same process ID")
	}

	if cpgrp1 == ppgrp {
		t.Fatalf("Parent and child 1 are in the same process group")
	}

	if cpid1 != cpgrp1 {
		t.Fatalf("Child 1's process group is not its process ID")
	}

	cmd2 := command(t)

	cmd2.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: cpgrp1}
	cmd2.Start()
	defer cmd2.Stop()

	cpid2, cpgrp2 := child(cmd2)

	if cpid2 == ppid {
		t.Fatalf("Parent and child 2 have the same process ID")
	}

	if cpgrp2 == ppgrp {
		t.Fatalf("Parent and child 2 are in the same process group")
	}

	if cpid2 == cpgrp2 {
		t.Fatalf("Child 2's process group is its process ID")
	}

	if cpid1 == cpid2 {
		t.Fatalf("Child 1 and 2 have the same process ID")
	}

	if cpgrp1 != cpgrp2 {
		t.Fatalf("Child 1 and 2 are not in the same process group")
	}
}

func TestForeground(t *testing.T) {
	signal.Ignore(syscall.SIGTTIN, syscall.SIGTTOU)

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		t.Logf("Can't test Foreground. Couldn't open /dev/tty: %s", err)
	}

	fpgrp := 0

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		tty.Fd(),
		syscall.TIOCGPGRP,
		uintptr(unsafe.Pointer(&fpgrp)))

	if errno != 0 {
		t.Fatalf("TIOCGPGRP failed with error code: %s", errno)
	}

	if fpgrp == 0 {
		t.Fatalf("Foreground process group is zero")
	}

	ppid, ppgrp := parent()

	cmd := command(t)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Ctty:       int(tty.Fd()),
		Foreground: true,
	}
	cmd.Start()

	cpid, cpgrp := child(cmd)

	if cpid == ppid {
		t.Fatalf("Parent and child have the same process ID")
	}

	if cpgrp == ppgrp {
		t.Fatalf("Parent and child are in the same process group")
	}

	if cpid != cpgrp {
		t.Fatalf("Child's process group is not the child's process ID")
	}

	cmd.Stop()

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
		tty.Fd(),
		syscall.TIOCSPGRP,
		uintptr(unsafe.Pointer(&fpgrp)))

	if errno != 0 {
		t.Fatalf("TIOCSPGRP failed with error code: %s", errno)
	}

	signal.Reset()
}
