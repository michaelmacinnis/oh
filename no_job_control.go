// Released under an MIT-style license. See LICENSE.

// +build !linux,!darwin,!dragonfly,!freebsd,!openbsd,!netbsd

package main

import "syscall"

func JobControlSupported() bool {
	return false
}

func SysProcAttr(group int) syscall.SysProcAttr{
	return &syscall.SysProcAttr{}
}
