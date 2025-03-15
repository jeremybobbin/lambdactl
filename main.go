package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"lambdactl/api"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sort"
	"strings"
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

// map[publickey]name
func GetLocalPublicKeys() (m map[string]string, err error) {
	home := os.Getenv("HOME")
	prefix := path.Join(home, ".ssh")
	human := strings.Replace(prefix, home, "~", 1)

	dir := os.DirFS(prefix)
	var paths []string
	paths, err = fs.Glob(dir, "*.pub")
	if err != nil || len(paths) == 0 {
		return nil, err
	}

	m = make(map[string]string, len(paths))
	for _, p := range paths {
		var key []byte
		key, err = fs.ReadFile(dir, p)
		if err != nil {
			continue
		}
		k, err := api.ParseKey(key)
		if err != nil {
			fmt.Println("get public keys", p, err)
			continue
		}
		m[k] = path.Join(human, p)
	}
	return
}

// map[publickey]name
func Intersection(m1, m2 map[string]string) (subset []string) {
	/*
		for k := range m2 {
			if _, ok := m1; ok {
				subset = append(subset, k)
			}
		}
	*/
	return
}

func Stretch(items [][]string, width int) []string {
	max := 0
	for _, columns := range items {
		max = Max(len(columns), max)
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

type MenuFn func (out chan string, rows []string) context.CancelFunc

func MenuClosure(ctx context.Context, stdin chan []byte, width, height int) MenuFn {
	return MenuFn(func (out chan string, rows []string) context.CancelFunc {
		ctx, cancel := context.WithCancel(ctx)
		keys := make(chan []byte)
		in := make(chan string)

		go func() {
			defer close(in)
			for _, row := range rows {
				select {
				case <-ctx.Done():
					return
				case in <- row:
				}
			}
		}()

		go func() {
			defer close(keys)
			for {
				select {
				case <-ctx.Done():
					return
				case key := <-stdin:
					keys <- key
				}
			}
		}()

		go func() {
			Menu(ctx, keys, out, os.Stderr, in, 10, width, height)
			close(out)
		}()

		return cancel
	})
}

func PromptCloudKeys(ctx context.Context, c *api.Client, ch chan string, menu MenuFn, width int) (map[string]string, context.CancelFunc, error) {
	keys, err, _ := c.SSHKeys()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get SSH keys: %s\n", err.Error())
	}

	for k, _ := range keys {
		if !strings.HasPrefix(k, "ssh-") || len(k) > 500 {
			continue
		}
	}

	items := make([][]string, 0, len(keys))
	local, _ := GetLocalPublicKeys()

	for k, v := range keys {
		for _, name := range v {
			if len(name) > 15 {
				continue
			}
			var path string
			if v, ok := local[k]; ok {
				path = v
			} else {
				path = "-"
			}
			item := [2]string{name, path}
			items = append(items, item[:])
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i][1] == "-" && items[j][1] != "-" {
			return false
		}
		return items[i][0] < items[j][0]
	})

	rows := Stretch(items, width)
	selections := make(map[string]string)
	for i, item := range items {
		selections[rows[i]] = item[0]
	}

	return selections, menu(ch, rows), nil
}

func Prompt(ctx context.Context, c *api.Client) error {
	tty, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open tty: %s\n", err.Error())
	}
	width, height, err := setup(tty)
	if err != nil {
		return fmt.Errorf("failed probing TTY size: %s\n", err.Error())
	}

	defer teardown(tty)

	stdin := make(chan []byte)
	go func() {
		defer close(stdin)
		var buf [4096]byte

		var err error
		for i, n := 0, 0; ; i += n {
			if i > len(buf)/2 {
				i = 0
			}
			n, err = tty.Read(buf[i:])
			if err != nil {
				return
			}
			stdin <- buf[i : i+n]
		}
	}()

	menu := MenuClosure(ctx, stdin, width, height)

	keys := make(chan string)
	selections, cancel, err := PromptCloudKeys(ctx, c, keys, menu, width)
	if err != nil {
		return err
	}

	var key string
	var ok bool
	for s := range keys {
		if key, ok = selections[s]; ok {
			cancel()
		}
	}

	quotes, titles, err := c.Availability()
	if err != nil {
		return fmt.Errorf("failed getting instance quotes: %s\n", err.Error())
	}

	sort.Slice(titles, func(i, j int) bool {
		return titles[i].Less(titles[j])
	})

	columns := make([]string, 3)
	items := make([][]string, len(titles))
	for i, title := range titles {
		q := quotes[title]
		r := title.Region()
		m := title.Model()
		items[i] = make([]string, len(columns))
		items[i][0] = m
		items[i][1] = r.String()
		items[i][2] = fmt.Sprintf("%5.2f", float32(q.PriceCentsPerHour)/100)

	}

	ch := make(chan string)
	rows := Stretch(items, width)
	selections2 := make(map[string]api.Title)
	for i, title := range titles {
		selections2[rows[i]] = title
	}

	cancel = menu(ch, rows)
	var title api.Title
	for s := range ch {
		var ok bool
		if title, ok = selections2[s]; ok {
			cancel()
		}
	}

	ids, err := c.Launch(title, "", []string{key}, nil, "")
	if err != nil {
		return fmt.Errorf("failed launching instance: %s\n", err.Error())
	}
	fmt.Println("got", ids)

	instances, err := c.Instances()
	fmt.Println(instances, err)

	return nil
}

func main() {
	c, err := api.NewClient(&http.Client{}, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to make API Client: %s\n", err.Error())
	}

	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)
	err = Prompt(ctx, c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	}
	fmt.Println(err)
}
