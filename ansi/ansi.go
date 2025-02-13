package ansi

import (
	"errors"
	"strings"
)

const (
	ESC        = '\033'
	CSI        = string(ESC) + "["
	ResetColor = CSI + "0m"
	Italic     = CSI + "3m"
	UnderLine  = CSI + "4m"
	BGblack    = CSI + "40m"
	BGred      = CSI + "41m"
	BGgreen    = CSI + "42m"
	BGblue     = CSI + "44m"
	BGwhite    = CSI + "47m"
	FGblack    = CSI + "30m"
	FGred      = CSI + "31m"
	FGgreen    = CSI + "32m"
	FGblue     = CSI + "34m"
	FGwhite    = CSI + "37m"
)

var ErrTooLong = errors.New("too long input string")

func SetColor(text string, Color ...string) string {
	ret := text + ResetColor
	for _, v := range Color {
		ret = v + ret
	}
	return ret
}

func CountAnsi(s string) int {
	count := 0
	inescape := false
	for _, k := range s {
		if k == ESC {
			inescape = true
		}
		if inescape {
			count++
		}
		if 0x40 <= k && k <= 0x7E {
			inescape = false
		}
	}
	return count
}

func MidleText(str string, length int, paddingChar ...string) (string, error) {
	strlen := len(str) - CountAnsi(str)
	if strlen > length {
		return "", ErrTooLong
	}
	paddingLen := (length - strlen) / 2
	paddingLen2 := (length - strlen) - paddingLen

	if len(paddingChar) == 0 {
		paddingChar = append(paddingChar, " ")
	}
	padding := strings.Repeat(paddingChar[0], int(paddingLen))
	padding2 := strings.Repeat(paddingChar[0], int(paddingLen2))
	return padding + str + padding2, nil
}

func drawLine(data []string, lengths []int) string {
	line := ""
	for k, v := range data {
		d, err := MidleText(v, lengths[k]-1)
		if errors.Is(err, ErrTooLong) {
			if lengths[k]-2 >= 0 {
				text := v[:lengths[k]-2]
				text += ">"
				d, _ = MidleText(text, lengths[k]-1)
			}
		}
		d += "|"
		line += d
	}
	return line + "\n"
}

func Table(titles []string, data [][]string, lengths []int, selected int) string {
	s := ""
	title := drawLine(titles, lengths)
	s += SetColor(title, BGwhite, FGblack)
	for k, v := range data {
		if k == selected {
			s += SetColor(drawLine(v, lengths), BGblue)
		} else {
			s += drawLine(v, lengths)
		}
	}
	return s
}
