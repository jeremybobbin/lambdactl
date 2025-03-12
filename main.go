package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
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

func main() {
	var err error
	items, max, err := ReadLines(os.Stdin)
	if err != nil {
		panic(err)
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open tty: %s\n", err.Error())
		os.Exit(1)
	}
	width, height, err := setup(tty)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to probe TTY size: %s\n", err.Error())
		os.Exit(1)
	}
	defer teardown(tty)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	Menu(ctx, tty, os.Stdout, os.Stderr, items, max, 10, width, height)
}
