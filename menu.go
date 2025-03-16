package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"os"
)

/*
#include <termios.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <errno.h>

static struct termios term[2];

int setup(int fd) {
	tcgetattr(fd, &term[0]);
	term[1] = term[0];
	term[1].c_iflag &= ~(BRKINT|PARMRK|ISTRIP|INLCR|IGNCR|ICRNL|IXON);
	term[1].c_lflag &= ~(ECHO|ECHONL|IEXTEN|ICANON); // |ICANON
	term[1].c_cflag &= ~(CSIZE|PARENB);
	term[1].c_cflag |= CS8;
	term[1].c_cc[VMIN] = 1;
	if (tcsetattr(fd, TCSANOW, &term[1]) < 0)
		return errno;
	return 0;
}

int dimensions(int fd, int *width, int *height) {
	struct winsize ws;
	if (ioctl(fd, TIOCGWINSZ, &ws) < 0) {
		return errno;
	}
	*width = ws.ws_col;
	*height = ws.ws_row;
	return 0;
}

int teardown(int fd) {
	return tcsetattr(fd, TCSANOW, &term[0]);
}
*/
import "C"

const (
	Padding = 2
)

func setup(f *os.File) error {
	r := C.setup(C.int(f.Fd()))
	if r != 0 {
		return fmt.Errorf("CGO setup failed - return code %d", r)
	}
	return nil
}

func dimensions(f *os.File) (int, int, error) {
	var err error
	var w, h C.int

	r := C.dimensions(C.int(f.Fd()), &w, &h)
	if r != 0 {
		err = fmt.Errorf("CGO dimensions failed - return code %d", r)
	}
	return int(w), int(h), err
}

func teardown(f *os.File) error {
	r := C.teardown(C.int(f.Fd()))
	if r != 0 {
		return fmt.Errorf("CGO failed - return code %d", r)
	}
	return nil
}

type Color int

type Fielder interface {
	Fields() []string
}

type Rower interface {
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
	buf := make([]rune, width-Padding)

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

	remaining := width - Padding
	for _, w := range widths {
		remaining -= w
	}

	rows := make([]string, len(items))
	var pad int
	if len(columns) > 1 {
		pad = remaining / (len(columns) - 1)
	} else {
		pad = remaining
	}
	for i, columns := range items {
		row := make([]string, len(columns))
		for j, column := range columns {
			if j == len(columns)-1 {
				row[j] = fmt.Sprintf("%*s", widths[j], column)
			} else {
				row[j] = fmt.Sprintf("%-*s", pad+widths[j], column)
			}
		}
		rows[i] = strings.Join(row, "")
	}

	return rows
}

type Dimensions struct {
	width, height int
}

func Menu(ctx context.Context, keys chan []byte, ch chan string, tty *os.File, w io.Writer, rows chan Rower, winch chan os.Signal, lines int) {
	var (
		width, height int
		err error
		sel, offset int
		items       []Rower
		display     = bufio.NewWriter(w)
		indicies    = make(map[string]int)
	)

	var input []rune
	width, height, err = dimensions(tty)
	if err != nil {
		fmt.Fprintf(os.Stderr, "menu err: %+v\n", err)
		return
	}

	// default colors
	fmt.Fprintf(display, "\x1b[0m")

loop:
	for {
		r := make([][]string, Min(lines, height))
		for i := 0; i < len(r);  {
			if i+offset >= len(items) {
				break
			}
			if f := items[i+offset].Fields(); f != nil {
				r[i] = f
				i++
			} else {
				j := i+offset
				delete(indicies, items[j].String())
				for j++; j < len(items); j++ {
					indicies[items[j].String()] = j-1
					items[j-1] = items[j]
				}
				items = items[:len(items)-1]
			}
		}
		strs := Stretch(r, width)

		for i, s := range strs {
			var color Color
			if i+offset == sel && i+offset < len(items) {
				color = Reverse
			}
			DrawLine(display, s, color, width)
		}

		// cursor up n-times, cursor to column n
		fmt.Fprintf(display, "\x1b[%dF\x1b[%dG", Min(lines, height), len(input)+1)

		display.Flush()
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
			if !ok {
				break loop
			}
		case <-winch:
			width, height, err = dimensions(tty)
			if err != nil {
				fmt.Fprintf(os.Stderr, "menu err: %+v\n", err)
				return
			}
			continue
		case <-ctx.Done():
			break loop
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
			fmt.Fprintf(display, "\x1b[G\x1b[J")
		case string(0x40 ^ 'D'), "\x1b":
			break loop
		case "\r":
			var item string
			if sel < 0 || sel >= len(items) {
				item = string(input)
			} else {
				item = items[sel].String()
			}
			select {
			case <-ctx.Done():
			case ch <- item:
			}
			break loop
		case string(0x40 ^ 'J'), string(0x40 ^ 'N'), "\x1bj", "\x1bn":
			sel++
			sel = Min(sel, len(items)-1)
			if sel >= offset+Min(lines, height) {
				offset++
			}
			offset = Min(offset, len(items)-Min(lines, height))
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
			offset = Max(0, len(items)-Min(lines, height))
			sel = Max(len(items)-1, 0)
		case string(0x40 ^ '?'), string(0x40 ^ 'H'):
			if len(input) == 0 {
				break
			}
			input = (input)[:len(input)-1]
			// backspace, space, backspace
			fmt.Fprintf(display, "\x08 \x08")
		default:
			for _, r := range string(key) {
				if strconv.IsGraphic(r) {
					display.Write([]byte(string(r)))
					input = append(input, r)
				}
			}
		}
	}

	fmt.Fprintf(display, "\x1b[G\x1b[J")
	display.Flush()
}
