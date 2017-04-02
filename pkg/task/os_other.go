// Released under an MIT license. See LICENSE.

// +build !linux,!darwin,!dragonfly,!freebsd,!openbsd,!netbsd,!solaris

package task

func initPlatformSpecific() {}

func initSignalHandling() {}

var evaluate = eval
