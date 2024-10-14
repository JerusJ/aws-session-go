package main

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func getAbsShell(s string) string {
	sAbs := s

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	home := usr.HomeDir

	sAbs = strings.ReplaceAll(s, "~", home)
	sAbs = os.ExpandEnv(sAbs)

	sAbs, err = filepath.Abs(sAbs)
	if err != nil {
		panic(err)
	}

	return sAbs
}

func isFile(f string) bool {
	if _, err := os.Stat(f); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}
