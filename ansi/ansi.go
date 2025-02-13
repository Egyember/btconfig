package ansi

import (
	"errors"
	"strings"
)

const (
	ESC        = '\033'
	CSI        = string(ESC) + "["
	ResetColor = CSI + "0m"
	BGblack    = CSI + "40m"
	BGred      = CSI + "41m"
	BGgreen    = CSI + "42m"
)

var ErrTooLong = errors.New("too long input string")

func SetColor(text string, Color ...string) string {
	ret := text + ResetColor
	for _, v := range Color {
		ret = v + ret
	}
	return ret
}

func countAnsi(s string) int {
	count := 0
	inescape := false
	for _, k := range s {
		if k == ESC {
			inescape = true
		}
		if 0x40 <= k && k <= 0x7E {
			inescape = false
		}
		if inescape {
			count++
		}
	}
	return count
}

func MidleText(str string, length int, paddingChar ...string) (string, error) {
	strlen := len(str) - countAnsi(str)
	if strlen > length {
		return "", ErrTooLong
	}
	paddingLen := (length - strlen) / 2

	if len(paddingChar) == 0 {
		paddingChar[0] = " "
	}
	padding := strings.Repeat(paddingChar[0], paddingLen)
	s := padding + str + padding
	return s, nil
}
