// Copied from https://golang.org/src/cmd/internal/browser/browser.go

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package browser provides utilities for interacting with users' browsers.
package browser

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/svenschwermer/gcal-notify/config"
)

// Commands returns a list of possible commands to use to open a url.
func Commands() [][]string {
	var cmds [][]string
	if exe := os.Getenv("BROWSER"); exe != "" {
		cmds = append(cmds, []string{exe})
	}
	switch runtime.GOOS {
	case "darwin":
		cmds = append(cmds, []string{"/usr/bin/open"})
	case "windows":
		cmds = append(cmds, []string{"cmd", "/c", "start"})
	default:
		if os.Getenv("DISPLAY") != "" {
			// xdg-open is only for use in a desktop environment.
			cmds = append(cmds, []string{"xdg-open"})
		}
	}
	cmds = append(cmds,
		[]string{"chrome"},
		[]string{"google-chrome"},
		[]string{"chromium"},
		[]string{"firefox"},
	)
	return cmds
}

// Open tries to open url in a browser and reports whether it succeeded.
func Open(url string) bool {
	if err := updateEnv(); err != nil {
		log.Printf("Failed to modify environment: %v", err)
	}
	for _, args := range Commands() {
		cmd := exec.Command(args[0], append(args[1:], url)...)
		config.Debug.Printf("browser: %v", cmd.Args)
		if cmd.Start() == nil && appearsSuccessful(cmd, 3*time.Second) {
			return true
		}
	}
	config.Debug.Printf("browser: opening %s failed", url)
	return false
}

// appearsSuccessful reports whether the command appears to have run successfully.
// If the command runs longer than the timeout, it's deemed successful.
// If the command runs within the timeout, it's deemed successful if it exited cleanly.
func appearsSuccessful(cmd *exec.Cmd, timeout time.Duration) bool {
	errc := make(chan error, 1)
	go func() {
		errc <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		return true
	case err := <-errc:
		return err == nil
	}
}

func updateEnv() error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("failed to connect to session bus: %w", err)
	}
	defer conn.Close()

	sysobj := conn.Object("org.freedesktop.systemd1", dbus.ObjectPath("/org/freedesktop/systemd1"))
	envVarint, err := sysobj.GetProperty("org.freedesktop.systemd1.Manager.Environment")
	if err != nil {
		return fmt.Errorf("failed to get systemd manager environment: %w", err)
	}
	var env []string
	err = envVarint.Store(&env)
	if err != nil {
		return fmt.Errorf("unexpected systemd manager environment type: %w", err)
	}
	for _, e := range env {
		if strings.HasPrefix(e, "DISPLAY=") || strings.HasPrefix(e, "WAYLAND_DISPLAY=") {
			kv := strings.SplitN(e, "=", 2)
			err = os.Setenv(kv[0], kv[1])
			if err != nil {
				return fmt.Errorf("failed to set environment variable %q: %w", e, err)
			}
		}
	}
	return nil
}
