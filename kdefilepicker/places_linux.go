// SPDX-License-Identifier: MIT

//go:build linux

package kdefilepicker

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type lsblkOutput struct {
	Blockdevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name       string        `json:"name"`
	Label      string        `json:"label"`
	Mountpoint string        `json:"mountpoint"`
	Fstype     string        `json:"fstype"`
	Size       string        `json:"size"`
	FSUse      string        `json:"fsuse%"`
	Children   []lsblkDevice `json:"children"`
}

func loadDevices() []place {
	out, err := exec.Command("lsblk", "-J", "-o", "NAME,LABEL,MOUNTPOINT,FSTYPE,SIZE,FSUSE%").Output()
	if err != nil {
		return nil
	}
	var root lsblkOutput
	if err := json.Unmarshal(out, &root); err != nil {
		return nil
	}
	var places []place
	var walk func([]lsblkDevice)
	walk = func(devs []lsblkDevice) {
		for _, d := range devs {
			if len(d.Children) > 0 {
				walk(d.Children)
			}
			if d.Fstype == "" || d.Fstype == "swap" || d.Fstype == "crypto_LUKS" {
				continue
			}
			name := d.Label
			if name == "" {
				continue
			}
			size := parseHumanSize(d.Size)
			used := uint64(0)
			mounted := d.Mountpoint != ""
			if mounted {
				var st syscall.Statfs_t
				if syscall.Statfs(d.Mountpoint, &st) == nil {
					size = st.Blocks * uint64(st.Bsize)
					used = (st.Blocks - st.Bfree) * uint64(st.Bsize)
				}
			}
			places = append(places, place{
				name:    name,
				path:    d.Mountpoint,
				icon:    "drive-harddisk",
				device:  true,
				used:    used,
				size:    size,
				mounted: mounted,
			})
		}
	}
	walk(root.Blockdevices)
	return places
}

func parseHumanSize(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	mult := uint64(1)
	switch s[len(s)-1] {
	case 'K':
		mult = 1 << 10
	case 'M':
		mult = 1 << 20
	case 'G':
		mult = 1 << 30
	case 'T':
		mult = 1 << 40
	case 'P':
		mult = 1 << 50
	}
	if mult != 1 {
		s = s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return uint64(f * float64(mult))
}
