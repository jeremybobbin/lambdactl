package main

import (
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

type MenuFn func(ctx context.Context, in chan StringerFielder, out chan string)

func MenuClosure(w io.Writer, stdin chan []byte, width, height int) MenuFn {
	return MenuFn(func(ctx context.Context, in chan StringerFielder, out chan string) {
		keys := make(chan []byte)

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

		defer close(out)
		Menu(ctx, keys, out, w, in, 10, width, height)
	})
}

type SSHKey struct {
	name string
	path *string
}

func (c SSHKey) String() string {
	return c.name
}

func (c SSHKey) Fields() []string {
	var fields = [2]string{
		c.name,
	}

	if c.path == nil {
		fields[1] = "-"
	} else {
		fields[1] = *c.path
	}
	return fields[:]
}

func Xor(a, b bool) bool {
	return (a && !b) || (!a && b)
}

func PromptCloudKeys(ctx context.Context, c *api.Client, ch chan string, menu MenuFn) (context.CancelFunc, error) {
	keys, err, _ := c.SSHKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH keys: %s\n", err.Error())
	}

	for k, _ := range keys {
		if !strings.HasPrefix(k, "ssh-") || len(k) > 500 {
			continue
		}
	}

	items := make([]SSHKey, 0, len(keys))
	local, _ := GetLocalPublicKeys()

	for k, v := range keys {
		for _, name := range v {
			if len(name) > 15 {
				continue
			}
			item := SSHKey{
				name: name,
			}
			if path, ok := local[k]; ok {
				item.path = &path
			}
			items = append(items, item)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if Xor(items[i].path == nil, items[j].path == nil) {
			return items[i].path != nil
		}
		return items[i].String() < items[j].String()
	})


	ctx, cancel := context.WithCancel(ctx)
	in := make(chan StringerFielder)

	go func() {
		defer close(in)
		for _, item := range items {
			select {
			case in <- item:
			case <-ctx.Done():
				return
			}
		}
	}()

	go menu(ctx, in, ch)
	return cancel, nil
}

type Quote struct {
	title api.Title
	price int
}

func (q Quote) String() string {
	return q.title.String()
}

func (q Quote) Fields() []string {
	return []string{
		q.title.Model(),
		q.title.Region().String(),
		fmt.Sprintf("%5.2f", float32(q.price)/100),
	}
}

func InstanceQuotes(ctx context.Context, c *api.Client, ch chan StringerFielder) error {
	quotes, _, err := c.Availability()
	if err != nil {
		return fmt.Errorf("failed getting instance quotes: %s\n", err.Error())
	}

	items := make([]Quote, 0, len(quotes))
	for title, quote := range quotes {
		items = append(items, Quote{
			title,
			quote.PriceCentsPerHour,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].price != items[j].price {
			return items[i].price < items[j].price
		}
		return items[i].title.Less(items[j].title)
	})

	go func() {
		defer close(ch)
		for _, item := range items {
			select {
			case ch <- item:
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func PromptCreateInstance(ctx context.Context, c *api.Client, menu MenuFn) error {
	quotes := make(chan StringerFielder)
	var fetch error
	go func() {
		fetch = InstanceQuotes(ctx, c, quotes)
	}()

	// Prompt Cloud Keys

	keys := make(chan string)
	cancel, err := PromptCloudKeys(ctx, c, keys, menu)
	if err != nil {
		return err
	}

	var key string
	for key = range keys {
		cancel()
	}

	// Prompt Instance Quote

	var title api.Title

	{
		ctx, cancel := context.WithCancel(ctx)
		titles := make(chan string)

		go menu(ctx, quotes, titles)

		for s := range titles {
			var err error
			title, err = api.ParseTitle(s)
			if err != nil {
				continue
			}
			cancel()
		}
	}

	if fetch != nil {
		return fmt.Errorf("failed fetching instance quotes: %s\n", fetch.Error())
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
	tty, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open tty: %s\n", err.Error())
	}
	width, height, err := setup(tty)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed probing TTY size: %s\n", err.Error())
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

	menu := MenuClosure(os.Stderr, stdin, width, height)

	c, err := api.NewClient(&http.Client{}, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to make API Client: %s\n", err.Error())
	}

	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)
	err = PromptCreateInstance(ctx, c, menu)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	}
	fmt.Println(err)
}
