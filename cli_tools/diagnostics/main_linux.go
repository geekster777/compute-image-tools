//  Copyright 2018 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func gatherSystemLogs(logs chan logFolder, errs chan error) {
	var commands = []runner{
		cmd{"uname", "-a", "os_version.txt", false},
		cmd{"lshw", "", "hardware.txt", false},
		cmd{"lscpu", "", "cpu.txt", false},
		cmd{"lspci", "-v", "pci_devices.txt", false},
		cmd{"lsblk", "-a", "block_devices.txt", false},
		cmd{"lsusb", "-v", "usb_devices.txt", false},
		cmd{"dmesg", "", "dmesg.txt", false},
	}

	logs <- logFolder{"System", runAll(commands, errs)}
}

func gatherDiskLogs(logs chan logFolder, errs chan error) {
	var commands = []runner{
		cmd{"fdisk", "-l", "fdisk.txt", false},
		cmd{"df", "", "df.txt", false},
		cmd{"mount", "", "mount.txt", false},
	}

	logs <- logFolder{"Disk", runAll(commands, errs)}
}

func gatherNetworkLogs(logs chan logFolder, errs chan error) {
	var commands = []runner{
		cmd{"ping", "-c 10 8.8.8.8", "ping.txt", false},
		cmd{"ifconfig", "-a", "ifconfig.txt", false},
	}

	logs <- logFolder{"Network", runAll(commands, errs)}
}

func gatherEventLogs(logs chan logFolder, errs chan error) {
	dirs := []string{"/var/log"}
	paths := make([]string, 0)
	// Recursively gather all the log files in a directory
	for len(dirs) > 0 {
		dir := dirs[0]
		dirs = dirs[1:]
		files, _ := ioutil.ReadDir(dir)
		for _, f := range files {
			path := filepath.Join(dir, f.Name())
			if f.IsDir() {
				dirs = append(dirs, path)
			} else {
				paths = append(paths, path)
			}
		}
	}
	logs <- logFolder{"Logs", paths}
}

func gatherLogs(trace bool) ([]logFolder, error) {
	runFuncs := []func(logs chan logFolder, errs chan error){
		gatherSystemLogs,
		gatherDiskLogs,
		gatherNetworkLogs,
		gatherEventLogs,
	}

	folderCount := len(runFuncs)
	folders := make([]logFolder, 0, folderCount)
	errStrings := make([]string, 0)
	ch := make(chan logFolder, folderCount)
	errs := make(chan error)

	for _, run := range runFuncs {
		go run(ch, errs)
	}

	for {
		select {
		case folder := <-ch:
			folders = append(folders, folder)
		case err := <-errs:
			errStrings = append(errStrings, err.Error())
		}

		if len(folders) == folderCount {
			break
		}
	}

	if len(errs) > 0 {
		return folders, errors.New(strings.Join(errStrings, "\n"))
	}
	return folders, nil
}
