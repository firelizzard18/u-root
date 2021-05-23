// Copyright 2012-2017 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !plan9

// mount mounts a filesystem at the specified path.
//
// Synopsis:
//     mount [-r] [-o options] [-t FSTYPE] DEV PATH
//
// Options:
//     -r: read only
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/mount/loop"
	"golang.org/x/sys/unix"
)

type mountOptions []string

func (o *mountOptions) String() string {
	return strings.Join(*o, ",")
}

func (o *mountOptions) Set(value string) error {
	for _, option := range strings.Split(value, ",") {
		*o = append(*o, option)
	}
	return nil
}

var (
	ro     = flag.Bool("r", false, "Read only mount")
	fsType = flag.String("t", "", "File system type")
	bind   = flag.Bool("bind", false, "Mount with -o bind")
	rbind  = flag.Bool("rbind", false, "Mount with -o bind,rec")

	makeShared      = flag.Bool("make-shared", false, "Mount with -o shared")
	makeSlave       = flag.Bool("make-slave", false, "Mount with -o slave")
	makePrivate     = flag.Bool("make-private", false, "Mount with -o private")
	makeUnbindable  = flag.Bool("make-unbindable", false, "Mount with -o unbindable")
	makeRShared     = flag.Bool("make-rshared", false, "Mount with -o shared,rec")
	makeRSlave      = flag.Bool("make-rslave", false, "Mount with -o slave,rec")
	makeRPrivate    = flag.Bool("make-rprivate", false, "Mount with -o private,rec")
	makeRUnbindable = flag.Bool("make-runbindable", false, "Mount with -o unbindable,rec")

	options mountOptions
)

func init() {
	flag.Var(&options, "o", "Comma separated list of mount options")
}

func loopSetup(filename string) (loopDevice string, err error) {
	loopDevice, err = loop.FindDevice()
	if err != nil {
		return "", err
	}
	if err := loop.SetFile(loopDevice, filename); err != nil {
		return "", err
	}
	return loopDevice, nil
}

// extended from boot.go
func getSupportedFilesystem(originFS string) ([]string, bool, error) {
	var known bool
	var err error
	fs, err := ioutil.ReadFile("/proc/filesystems")
	if err != nil {
		return nil, known, err
	}
	var returnValue []string
	for _, f := range strings.Split(string(fs), "\n") {
		n := strings.Fields(f)
		last := len(n) - 1
		if last < 0 {
			continue
		}
		if n[last] == originFS {
			known = true
		}
		returnValue = append(returnValue, n[last])
	}
	return returnValue, known, err

}

func informIfUnknownFS(originFS string) {
	knownFS, known, err := getSupportedFilesystem(originFS)
	if err != nil {
		// just don't make things even worse...
		return
	}
	if !known {
		log.Printf("Hint: unknown filesystem %s. Known are: %v", originFS, knownFS)
	}
}

func main() {
	n := []string{"/proc/self/mounts", "/proc/mounts", "/etc/mtab"}
	for _, p := range n {
		if b, err := ioutil.ReadFile(p); err == nil {
			fmt.Print(string(b))
			os.Exit(0)
		}
	}

	flag.Parse()
	if len(flag.Args()) < 2 {
		flag.Usage()
		os.Exit(1)
	}
	a := flag.Args()
	dev := a[0]
	var path string
	if len(a) > 1 {
		path = a[1]
	}
	var flags uintptr
	var data []string
	var err error
	for _, option := range options {
		switch option {
		case "loop":
			dev, err = loopSetup(dev)
			if err != nil {
				log.Fatal("Error setting loop device:", err)
			}
		default:
			if f, ok := opts[option]; ok {
				flags |= f
			} else {
				data = append(data, option)
			}
		}
	}
	if *ro {
		flags |= unix.MS_RDONLY
	}
	if *bind || *rbind {
		flags |= unix.MS_BIND
	}
	if *makeShared || *makeRShared {
		flags |= unix.MS_SHARED
	}
	if *makeSlave || *makeRSlave {
		flags |= unix.MS_SLAVE
	}
	if *makePrivate || *makeRPrivate {
		flags |= unix.MS_PRIVATE
	}
	if *makeUnbindable || *makeRUnbindable {
		flags |= unix.MS_UNBINDABLE
	}
	if *rbind || *makeRShared || *makeRSlave || *makeRPrivate || *makeRUnbindable {
		flags |= unix.MS_BIND | unix.MS_REC
	}
	if *fsType == "" {
		if _, err := mount.TryMount(dev, path, strings.Join(data, ","), flags); err != nil {
			log.Fatalf("%v", err)
		}
	} else {
		if _, err := mount.Mount(dev, path, *fsType, strings.Join(data, ","), flags); err != nil {
			log.Printf("%v", err)
			informIfUnknownFS(*fsType)
			os.Exit(1)
		}
	}
}
