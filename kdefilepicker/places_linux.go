// SPDX-License-Identifier: MIT

//go:build linux

package kdefilepicker

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "lsblk", "-J", "-o", "NAME,LABEL,MOUNTPOINT,FSTYPE,SIZE,FSUSE%").Output()
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
				name:   name,
				path:   d.Mountpoint,
				icon:   "drive-harddisk",
				device: true,
				used:   used,
				size:   size,
			})
		}
	}
	walk(root.Blockdevices)
	return places
}

func parseHumanSize(s string) uint64 {
	s = strings.ToUpper(strings.TrimSpace(s))
	if s == "" {
		return 0
	}
	// lsblk emits either "512B", "1.5G" or occasionally "1.5GiB".
	s = strings.TrimSuffix(s, "IB")
	s = strings.TrimSuffix(s, "B")
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
	if err != nil || f <= 0 {
		return 0
	}
	if f >= float64(^uint64(0))/float64(mult) {
		return 0
	}
	return uint64(f * float64(mult))
}
