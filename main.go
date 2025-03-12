package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
)

/*
#include <termios.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <errno.h>

static struct termios term[2];

int setup(int fd, int *width, int *height) {
	struct winsize ws;
	int n;
		if ((n = ioctl(fd, TIOCGWINSZ, &ws)) < 0) {
			return errno;
	}
		*width = ws.ws_col;
		*height = ws.ws_row;

		tcgetattr(fd, &term[0]);
		term[1] = term[0];
		term[1].c_iflag &= ~(BRKINT|PARMRK|ISTRIP|INLCR|IGNCR|ICRNL|IXON);
		term[1].c_lflag &= ~(ECHO|ECHONL|IEXTEN|ICANON); // |ICANON
		term[1].c_cflag &= ~(CSIZE|PARENB);
		term[1].c_cflag |= CS8;
		term[1].c_cc[VMIN] = 1;
		if (tcsetattr(fd, TCSANOW, &term[1]) < 0) {
		return errno;
	}
}

int teardown(int fd) {
	return tcsetattr(fd, TCSANOW, &term[0]);
}
*/
import "C"

func setup(f *os.File) (int, int, error) {
	var err error
	var w, h C.int

	r := C.setup(C.int(f.Fd()), &w, &h)
	if r != 0 {
		err = fmt.Errorf("CGO failed - return code %d", r)
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

const (
	Normal Color = iota
	Reverse
)

var (
	width, height int
	max           int
	lines         int = 10
	prompt        string
	matches       []string
	items         []string
	text          string
	sel, offset   int
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

func DrawMenu(w io.Writer, items []string, width, max int) {
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

func ReadLines(r io.Reader) (lines []string, max int, err error) {
	scanner := bufio.NewScanner(r)

	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		if n := len(line); n > 0 && line[n-1] == '\n' {
			line = line[:n-1]
		}
		max = Max(Width(line), max)
		lines = append(lines, line)
	}

	err = scanner.Err()
	return
}

func handle(key string, input *[]rune, out, err io.Writer) (end bool) {
	switch key {
	case "\x1b\n", "\x1b\r":
		// cursor to start of line, clear rest of line
		fmt.Fprintf(out, "%s\n", items[sel])
		fmt.Fprintf(err, "\x1b[G\x1b[J")
	case string(0x40 ^ 'D'), "\x1b":
		end = true
	case "\r":
		fmt.Fprintf(os.Stdout, "%s\n", items[sel])
		end = true
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
		if len(*input) == 0 {
			break
		}
		*input = (*input)[:len(*input)-1]
		// backspace, space, backspace
		fmt.Fprintf(err, "\x08 \x08")
	default:
		for _, r := range key {
			if strconv.IsGraphic(r) {
				err.Write([]byte(string(r)))
				*input = append(*input, r)
			}
		}
	}
	return
}

func main() {
	var err error
	items, max, err = ReadLines(os.Stdin)
	if err != nil {
		panic(err)
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open tty: %s\n", err.Error())
		os.Exit(1)
	}
	width, height, err = setup(tty)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to probe TTY size: %s\n", err.Error())
		os.Exit(1)
	}
	defer teardown(tty)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	keys := make(chan string)

	pr, pw := io.Pipe()
	go io.Copy(pw, tty)

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

	stderr := bufio.NewWriter(os.Stderr)
	stdout := bufio.NewWriter(os.Stdout)

	var input []rune

	// default colors
	fmt.Fprintf(stderr, "\x1b[0m")

	for {
		DrawMenu(stderr, items[offset:], width, max)

		// cursor up n-times, cursor to column n
		fmt.Fprintf(stderr, "\x1b[%dF\x1b[%dG", Min(lines, len(items)), len(input)+1)

		stderr.Flush()
		if key, ok := <-keys; !ok {
			break
		} else if handle(key, &input, stdout, stderr) {
			cancel()
		}
		stdout.Flush()
	}

	stderr.Flush()

	// cursor to column 1, clear everything after the cursor
	fmt.Fprintf(os.Stderr, "\x1b[G\x1b[J")
}
