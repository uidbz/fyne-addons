// SPDX-License-Identifier: MIT

//go:build !linux && !windows

package kdefilepicker

func loadDevices() []place { return nil }
