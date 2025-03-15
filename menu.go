package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Color int

type Fielder interface {
	Fields() []string
}

type StringerFielder interface {
	Fielder
	fmt.Stringer
}

const (
	Normal Color = iota
	Reverse
)

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Width(str string) (n int) {
	for _, c := range str {
		if c == '\t' {
			n += 8
			continue
		}
		n += 1
	}
	return
}

func DrawLine(w io.Writer, t string, col Color, width int) {
	padding := 2
	buf := make([]rune, width-padding)

	text := []rune(t)

	for i, j, n := 0, 0, 0; i < len(buf); i++ {
		if n > 0 {
			n--
			buf[i] = ' '
		} else if j < len(text) {
			switch text[j] {
			case '\t':
				n = 7
				buf[i] = ' '
				j++
			default:
				buf[i] = text[j]
				j++
			}
		} else {
			buf[i] = ' '
		}
	}

	switch col {
	case Reverse:
		// cursor column n
		fmt.Fprintf(w, "\n\x1b[2K\x1b[7m %s \x1b[0m", string(buf))
	case Normal:
		fmt.Fprintf(w, "\n\x1b[2K %s ", string(buf))
	}

}

func Stretch(items [][]string, width int) []string {
	max := 0
	for i := range items {
		max = Max(len(items[i]), max)
	}

	columns := make([]string, max)
	widths := make([]int, len(columns))
	for i := range items {
		for j := range items[i] {
			widths[j] = Max(widths[j], len(items[i][j]))
		}
	}

	remaining := width
	for _, w := range widths {
		remaining -= w
	}

	rows := make([]string, len(items))
	pad := remaining / (len(columns) - 1)
	for i, columns := range items {
		row := make([]string, len(columns))
		for j, column := range columns {
			if j == len(columns)-1 {
				row[j] = fmt.Sprintf("%*s", widths[j], column)
			} else {
				row[j] = fmt.Sprintf("%-*s", pad, column)
			}
		}
		rows[i] = strings.Join(row, "")
	}

	return rows
}

func Menu(ctx context.Context, keys chan []byte, ch chan string, Stderr io.Writer, rows chan StringerFielder, lines, width, height int) {
	ctx, cancel := context.WithCancel(ctx)

	var (
		sel, offset int
		items       []StringerFielder
	)

	lines = Min(height, lines)

	stderr := bufio.NewWriter(Stderr)
	indicies := make(map[string]int)

	var input []rune

	// default colors
	fmt.Fprintf(stderr, "\x1b[0m")

	for {
		r := make([][]string, Min(lines, len(items)))
		for i := range r {
			if i + offset >= len(items) {
				break
			}
			r[i] = items[i+offset].Fields()
		}
		strs := Stretch(r, width)

		for i, s := range strs {
			var color Color
			if i+offset == sel {
				color = Reverse
			}
			DrawLine(stderr, s, color, width)
		}

		// cursor up n-times, cursor to column n
		fmt.Fprintf(stderr, "\x1b[%dF\x1b[%dG", Min(lines, len(items)), len(input)+1)

		stderr.Flush()
		var key []byte
		var ok bool
		select {
		case item, ok := <-rows:
			if !ok {
				rows = nil
				continue
			}
			id := item.String()
			if i, ok := indicies[item.String()]; ok {
				items[i] = item
			} else {
				indicies[id] = len(items)
				items = append(items, item)
			}
			continue
		case key, ok = <-keys:
		}
		if !ok {
			break
		}

		switch string(key) {
		case "\x1b\n", "\x1b\r":
			// cursor to start of line, clear rest of line
			var item string
			if sel < 0 || sel >= len(items)-1 {
				item = string(input)
			} else {
				item = items[sel].String()
			}
			select {
			case <-ctx.Done():
			case ch <- item:
			}
			fmt.Fprintf(stderr, "\x1b[G\x1b[J")
		case string(0x40 ^ 'D'), "\x1b":
			cancel()
		case "\r":
			var item string
			if sel < 0 || sel >= len(items)-1 {
				item = string(input)
			} else {
				item = items[sel].String()
			}
			select {
			case <-ctx.Done():
			case ch <- item:
			}
			cancel()
		case string(0x40 ^ 'J'), string(0x40 ^ 'N'), "\x1bj", "\x1bn":
			sel++
			sel = Min(sel, len(items)-1)
			if sel >= offset+lines {
				offset++
			}
			offset = Min(offset, len(items)-lines)
			offset = Max(offset, 0)
		case string(0x40 ^ 'K'), "\x1bk", string(0x40 ^ 'P'), "\x1bp":
			if sel-1 <= 0 {
				sel = 0
				break
			}
			sel--
			if offset <= 0 {
				break
			}
			if sel <= offset {
				offset--
			}

		case string(0x40 ^ 'G'), "\x1bg":
			offset = 0
			sel = 0
		case "\x1bG":
			offset = Max(0, len(items)-lines)
			sel = Max(len(items)-1, 0)
		case string(0x40 ^ '?'), string(0x40 ^ 'H'):
			if len(input) == 0 {
				break
			}
			input = (input)[:len(input)-1]
			// backspace, space, backspace
			fmt.Fprintf(stderr, "\x08 \x08")
		default:
			for _, r := range string(key) {
				if strconv.IsGraphic(r) {
					stderr.Write([]byte(string(r)))
					input = append(input, r)
				}
			}
		}
	}

	stderr.Flush()

	// cursor to column 1, clear everything after the cursor
	fmt.Fprintf(os.Stderr, "\x1b[G\x1b[J")
}
