// Released under an MIT-style license. See LICENSE.

// +build !linux,!darwin,!dragonfly,!freebsd,!openbsd,!netbsd

package main

// TODO: !solaris should also be in the list above.

import "syscall"

func InitSignalHandling() {}

func InputIsTTY() bool {
	// TODO: Not sure what to do on non-Unix platforms.
	return false
}

func JobControlSupported() bool {
	return false
}

func SetProcessGroup() int {
	// TODO: Not sure what to do on non-Unix platforms.
	return 0
}

func SysProcAttr(group int) syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
