package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
)

type Color int

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

func DrawLine(w io.Writer, t string, col Color, width, max int) {
	padding := 2
	buf := make([]rune, Min(width-padding, max+padding))

	text := []rune(t)

	for i, j, n := 0, 0, 0; i < len(buf); i++ {
		if n > 0 {
			n--
			buf[i] = ' '
		} else if i+3 >= width-padding {
			buf[i] = '.'
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

/*
func DrawMenu(w io.Writer, items []string, width, lines, offset, max int) {
	for n := 0; n < lines; n++ {
		var item string
		var color Color
		if n < len(items) {
			item = items[n]
		}
		if n+offset == sel {
			color = Reverse
		}

		if n < len(items) {
			DrawLine(w, item, color, width, max)
		}
	}
}
*/

func Menu(ctx context.Context, Stdin io.Reader, Stdout, Stderr io.Writer, items []string, max, lines, width, height int) {
	ctx, cancel := context.WithCancel(ctx)
	keys := make(chan string)

	pr, pw := io.Pipe()
	go io.Copy(pw, Stdin)

	var (
		sel, offset   int
		err error
	)


	go func() {
		defer close(keys)
		var buf [8]byte
		var n int
		for {
			n, err = pr.Read(buf[:])
			if err != nil {
				return
			}
			keys <- string(buf[:n])
		}
	}()

	go func() {
		<-ctx.Done()
		pw.Close()
	}()

	stderr := bufio.NewWriter(Stdout)
	stdout := bufio.NewWriter(Stderr)

	var input []rune

	// default colors
	fmt.Fprintf(stderr, "\x1b[0m")

	for {
		//DrawMenu(stderr, items[offset:], width, lines, offset, max)
		for n := 0; n < lines; n++ {
			var item string
			var color Color
			if n < len(items) {
				item = items[n]
			}
			if n+offset == sel {
				color = Reverse
			}

			if n < len(items) {
				DrawLine(stderr, item, color, width, max)
			}
		}

		// cursor up n-times, cursor to column n
		fmt.Fprintf(stderr, "\x1b[%dF\x1b[%dG", Min(lines, len(items)), len(input)+1)

		stderr.Flush()
		key, ok := <-keys
		if !ok {
			break
		}

		switch key {
		case "\x1b\n", "\x1b\r":
			// cursor to start of line, clear rest of line
			fmt.Fprintf(stdout, "%s\n", items[sel])
			fmt.Fprintf(stderr, "\x1b[G\x1b[J")
		case string(0x40 ^ 'D'), "\x1b":
			cancel()
		case "\r":
			fmt.Fprintf(os.Stdout, "%s\n", items[sel])
			cancel()
		case string(0x40 ^ 'J'), string(0x40 ^ 'N'), "\x1bj", "\x1bn":
			if sel >= len(items)-1 {
				sel = Max(0, len(items)-1)
				break
			}
			sel++
			if sel+lines >= len(items) {
				break
			}
			if sel >= offset+1+(lines/2) {
				offset++
			}
		case string(0x40 ^ 'K'), "\x1bk", string(0x40 ^ 'P'), "\x1bp":
			if sel <= 0 {
				sel = 0
				break
			}
			sel--
			if offset <= 0 {
				break
			}
			if sel <= offset+(lines/2)-1 {
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
			for _, r := range key {
				if strconv.IsGraphic(r) {
					stderr.Write([]byte(string(r)))
					input = append(input, r)
				}
			}
		}

		stdout.Flush()
	}

	stderr.Flush()

	// cursor to column 1, clear everything after the cursor
	fmt.Fprintf(os.Stderr, "\x1b[G\x1b[J")
}
