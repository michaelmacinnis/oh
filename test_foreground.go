package main

import (
	"io"
        "os/exec"
        "syscall"
	"testing"
	"unsafe"
)

//#include <signal.h>
//#include <unistd.h>
//void ignore(void) {
//      signal(SIGTTOU, SIG_IGN);
//      signal(SIGTTIN, SIG_IGN);
//}
import "C"

type test *testing.T

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
		println("Failed!")
		//cmd.Fatal(err)
	}

	return
}

func command(t *testing.T) *handle {
        cmd := exec.Command("cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		println("Failed!")
		//t.Fatal(err)
	}

	return &handle{cmd, stdin, test(t)}
}

func parent() (pid, pgrp int) {
	return syscall.Getpid(), syscall.Getpgrp()
}

func TestForeground(t *testing.T) {
	fpgrp := 0

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
				       uintptr(syscall.Stdout),
				       syscall.TIOCGPGRP,
				       uintptr(unsafe.Pointer(&fpgrp)))


	if errno != 0 || fpgrp == 0 {
		println("Failed!")
		//t.FailNow()
	}

	ppid, ppgrp := parent()

	cmd := command(t)

        cmd.SysProcAttr = &syscall.SysProcAttr{Foreground: true}
	cmd.Start()

	cpid, cpgrp := child(cmd)

	if cpid == ppid || cpgrp == ppgrp || cpid != cpgrp {
		println("Failed!")
		//t.FailNow()
	}

	cmd.Stop()

	syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdout),
			syscall.TIOCSPGRP, uintptr(unsafe.Pointer(&fpgrp)))
}

func main() {
	C.ignore()

	TestForeground(nil)
}
