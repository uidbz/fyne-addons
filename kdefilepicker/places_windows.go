// SPDX-License-Identifier: MIT

//go:build windows

package kdefilepicker

import (
	"syscall"
	"unsafe"
)

var (
	modMpr                  = syscall.NewLazyDLL("mpr.dll")
	modKernel32             = syscall.NewLazyDLL("kernel32.dll")
	procWNetGetConnectionW  = modMpr.NewProc("WNetGetConnectionW")
	procGetLogicalDrives    = modKernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW       = modKernel32.NewProc("GetDriveTypeW")
	procGetVolumeInfoW      = modKernel32.NewProc("GetVolumeInformationW")
	procGetDiskFreeSpaceExW = modKernel32.NewProc("GetDiskFreeSpaceExW")
)

// Win32 drive types (WINBASE.h).
const (
	driveFixed  = 3
	driveRemote = 4
)

// loadDevices enumerates logical drives: fixed drives get their volume
// label and a capacity bar, mapped network drives are named after their
// UNC share. GetVolumeInformationW is deliberately only called for fixed
// drives: on remote or empty removable roots it can block for seconds.
func loadDevices() []place {
	mask, _, _ := procGetLogicalDrives.Call()
	var places []place
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		letter := string(rune('A' + i))
		root := letter + `:\`
		p := place{path: root, mounted: true}
		switch driveType(root) {
		case driveRemote:
			p.icon = "network-workgroup"
			p.name = uncShare(letter + ":")
			if p.name == "" {
				p.name = letter + ":"
			}
		case driveFixed:
			p.icon = "drive-harddisk"
			p.name = volumeLabel(root)
			if p.name == "" {
				p.name = letter + ":"
			}
			if total, free, ok := diskFree(root); ok && total > 0 {
				p.size = total
				p.used = total - free
			}
		default:
			// Removable/media drives: list them without querying, like
			// KDE does for unmounted devices.
			p.icon = "drive-harddisk"
			p.name = letter + ":"
		}
		places = append(places, p)
	}
	return places
}

func driveType(root string) int {
	r, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(root))))
	return int(r)
}

// uncShare resolves a mapped drive letter to its UNC path.
func uncShare(local string) string {
	var buf [4096]uint16
	size := uint32(len(buf))
	r, _, _ := procWNetGetConnectionW.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(local))),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r != 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:])
}

func volumeLabel(root string) string {
	var label [261]uint16
	r, _, _ := procGetVolumeInfoW.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(root))),
		uintptr(unsafe.Pointer(&label[0])),
		uintptr(len(label)),
		0, 0, 0, 0, 0,
	)
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(label[:])
}

func diskFree(root string) (total, free uint64, ok bool) {
	var avail, totalBytes, totalFree uint64
	r, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(root))),
		uintptr(unsafe.Pointer(&avail)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if r == 0 {
		return 0, 0, false
	}
	return totalBytes, totalFree, true
}
