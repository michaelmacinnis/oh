package main

import (
	"io"
        "os/exec"
        "syscall"
	"testing"
)

func child(t *testing.T, cmd *exec.Cmd) (pid, pgrp int) {
	pid = cmd.Process.Pid

	pgrp, err := syscall.Getpgid(pid)
	if err != nil {
		t.Fatal(err)
	}

	return
}

func parent(t *testing.T) (pid, pgrp int) {
	return syscall.Getpid(), syscall.Getpgrp()
}

func command(t *testing.T) (cmd *exec.Cmd, stdin io.WriteCloser) {
        cmd = exec.Command("cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	return
}

func stop(t *testing.T, cmd *exec.Cmd, stdin io.WriteCloser) {
}

func TestBare(t *testing.T) {
	ppid, ppgrp := parent(t)

	cmd, stdin := command(t)

	cmd.Start()

	cpid, cpgrp := child(t, cmd)

	if cpid == ppid || cpgrp != ppgrp {
		t.FailNow()
	}

	stdin.Close()
	cmd.Wait()
}

func TestSetpgid(t *testing.T) {
	ppid, ppgrp := parent(t)

	cmd, stdin := command(t)

        cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Start()

	cpid, cpgrp := child(t, cmd)

	if cpid == ppid || cpgrp == ppgrp || cpid != cpgrp {
		t.FailNow()
	}

	stdin.Close()
	cmd.Wait()
}

func TestJoinpgrp(t *testing.T) {
	ppid, ppgrp := parent(t)

	cmd1, stdin1 := command(t)

        cmd1.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd1.Start()

	cpid1, cpgrp1 := child(t, cmd1)

	if cpid1 == ppid || cpgrp1 == ppgrp || cpid1 != cpgrp1 {
		t.FailNow()
	}

	cmd2, stdin2 := command(t)

        cmd2.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Joinpgrp: cpgrp1}
	cmd2.Start()

	cpid2, cpgrp2 := child(t, cmd2)

	if cpid2 == ppid || cpgrp2 == ppgrp || cpid2 == cpgrp2 {
		t.FailNow()
	}

	if cpid1 == cpid2 || cpgrp1 != cpgrp2 {
		t.FailNow()
	}

	stdin1.Close()
	cmd1.Wait()

	stdin2.Close()
	cmd2.Wait()
}

func TestJoinpgrpImpliedSetpgid(t *testing.T) {
	ppid, ppgrp := parent(t)

	cmd1, stdin1 := command(t)

        cmd1.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd1.Start()

	cpid1, cpgrp1 := child(t, cmd1)

	if cpid1 == ppid || cpgrp1 == ppgrp || cpid1 != cpgrp1 {
		t.FailNow()
	}

	cmd2, stdin2 := command(t)

        cmd2.SysProcAttr = &syscall.SysProcAttr{Joinpgrp: cpgrp1}
	cmd2.Start()

	cpid2, cpgrp2 := child(t, cmd2)

	if cpid2 == ppid || cpgrp2 == ppgrp || cpid2 == cpgrp2 {
		t.FailNow()
	}

	if cpid1 == cpid2 || cpgrp1 != cpgrp2 {
		t.FailNow()
	}

	stdin1.Close()
	cmd1.Wait()

	stdin2.Close()
	cmd2.Wait()
}

