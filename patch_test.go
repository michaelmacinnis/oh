package main

import (
	"io"
        "os/exec"
        "syscall"
	"testing"
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

	cpid, cpgrp := child(cmd)

	if cpid == ppid || cpgrp != ppgrp {
		t.FailNow()
	}

	cmd.Stop()
}

func TestSetpgid(t *testing.T) {
	ppid, ppgrp := parent()

	cmd := command(t)

        cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Start()

	cpid, cpgrp := child(cmd)

	if cpid == ppid || cpgrp == ppgrp || cpid != cpgrp {
		t.FailNow()
	}

	cmd.Stop()
}

func TestJoinpgrp(t *testing.T) {
	ppid, ppgrp := parent()

	cmd1 := command(t)

        cmd1.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd1.Start()

	cpid1, cpgrp1 := child(cmd1)

	if cpid1 == ppid || cpgrp1 == ppgrp || cpid1 != cpgrp1 {
		t.FailNow()
	}

	cmd2 := command(t)

        cmd2.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Joinpgrp: cpgrp1}
	cmd2.Start()

	cpid2, cpgrp2 := child(cmd2)

	if cpid2 == ppid || cpgrp2 == ppgrp || cpid2 == cpgrp2 {
		t.FailNow()
	}

	if cpid1 == cpid2 || cpgrp1 != cpgrp2 {
		t.FailNow()
	}

	cmd1.Stop()
	cmd2.Stop()
}

func TestJoinpgrpImpliedSetpgid(t *testing.T) {
	ppid, ppgrp := parent()

	cmd1 := command(t)

        cmd1.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd1.Start()

	cpid1, cpgrp1 := child(cmd1)

	if cpid1 == ppid || cpgrp1 == ppgrp || cpid1 != cpgrp1 {
		t.FailNow()
	}

	cmd2 := command(t)

        cmd2.SysProcAttr = &syscall.SysProcAttr{Joinpgrp: cpgrp1}
	cmd2.Start()

	cpid2, cpgrp2 := child(cmd2)

	if cpid2 == ppid || cpgrp2 == ppgrp || cpid2 == cpgrp2 {
		t.FailNow()
	}

	if cpid1 == cpid2 || cpgrp1 != cpgrp2 {
		t.FailNow()
	}

	cmd1.Stop()
	cmd2.Stop()
}

