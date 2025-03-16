package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"lambdactl/api"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"sort"
	"strings"
	"time"
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

func MenuClosure(outer context.Context, w io.Writer) (context.Context, MenuFn) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open tty: %s\n", err.Error())
	}
	width, height, err := setup(tty)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed probing TTY size: %s\n", err.Error())
	}


	inner, cancel := context.WithCancel(context.Background())
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
			select {
			case stdin <- buf[i : i+n]:
			case <-outer.Done():
				return
			}
		}
	}()

	go func() {
		defer cancel()
		defer tty.Close()
		defer teardown(tty)
		<-outer.Done()
	}()


	return inner, MenuFn(func(ctx context.Context, in chan StringerFielder, out chan string) {
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

		Menu(ctx, keys, out, w, in, 10, width, height)
		close(out)
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

type Basic string

func (b Basic) String() string {
	return string(b)
}

func (b Basic) Fields() []string {
	return []string{string(b)}
}

func MenuBasic(ctx context.Context, values []string, menu MenuFn) string {
	in := make(chan StringerFielder)
	out := make(chan string)

	go func() {
		defer close(in)
		for _, s := range values {
			select {
			case in <- Basic(s):
			case <-ctx.Done():
				return
			}
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go menu(ctx, in, out)
	return <-out
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

type Instance struct {
	instance api.Instance
}

func (i *Instance) String() string {
	return i.instance.ID
}

func (i *Instance) Fields() []string {
	instance := i.instance
	var name, ip, keys string
	if p := instance.Name; p != nil {
		name = *p
	}
	if p := instance.IP; p != nil {
		ip = *p
	}

	if p := instance.SSHKeyNames; len(p) > 0 {
		keys = strings.Join(p, ", ")
	}

	fields := []string{
		name,
		instance.InstanceQuote.Name,
		instance.Region.Description,
		ip,
		keys,
		instance.Status.String(),
		fmt.Sprintf("%5.2f", float32(instance.InstanceQuote.PriceCentsPerHour)/100),
	}

	for i := range fields {
		if fields[i] == "" {
			fields[i] = "-"
		}
	}

	return fields
}

func Instances(ctx context.Context, c *api.Client, ch chan StringerFielder) error {
	instances, err := c.Instances()
	if err != nil {
		return err
	}

	items := make([]Instance, 0, len(instances))
	for i := range instances {
		items = append(items, Instance{
			instances[i],
		})
	}

	sort.Slice(items, func(i, j int) bool {
		a, b := items[i].instance, items[j].instance
		return a.ID < b.ID
	})

	go func() {
		for i := range items {
			select {
			case ch <- &items[i]:
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
			title, err = api.ParseTitle(s)
			if err != nil {
				continue
			}
			cancel()
		}
	}

	if err != nil {
		return err
	}

	if fetch != nil {
		return fmt.Errorf("failed fetching instance quotes: %s\n", fetch.Error())
	}

	_, err = c.Launch(title, "", []string{key}, nil, "")
	if err != nil {
		return fmt.Errorf("failed launching instance: %s\n", err.Error())
	}

	//_, err := c.Instances()

	return nil
}

func PromptSSH(ctx context.Context, c *api.Client, menu MenuFn) (*exec.Cmd,error) {
	instances, err := PromptInstances(ctx, c, menu)
	if err != nil {
		return nil, err
	}

	if len(instances) < 1 {
		return nil, fmt.Errorf("no instances chosen")
	}
	instance := instances[0]

	if instance.instance.IP == nil {
		return nil, fmt.Errorf("chosen instance has no IP")
	}

	cmd := exec.Command("ssh", "ubuntu@"+*instance.instance.IP)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, nil
}

func PromptTerminate(ctx context.Context, c *api.Client, menu MenuFn) (error) {
	instances, err := PromptInstances(ctx, c, menu)
	if err != nil {
		return err
	}
	var ids []string
	for i := range instances {
		ids = append(ids, instances[i].instance.ID)
	}
	c.Terminate(ids)
	return nil
}

type NilFielder string

func (nf NilFielder) String() string {
	return string(nf)
}

func (_ NilFielder) Fields() []string {
	return nil
}

func PromptInstances(ctx context.Context, c *api.Client, menu MenuFn) ([]Instance, error) {
	ch := make(chan StringerFielder)

	var fetch error
	instances := make(map[string]*Instance)
	go func() {
		defer close(ch)
		for i := 0; ; i++ {

			var latest []api.Instance
			latest, fetch = c.Instances()
			if fetch != nil {
				return
			}

			set := make(map[string]*api.Instance, len(latest))
			ptrs := make([]*api.Instance, len(latest))
			for i := range latest {
				set[latest[i].ID] = &latest[i]
				ptrs[i] = &latest[i]
			}

			for id := range instances {
				if _, ok := set[id]; ok {
					continue
				}

				delete(instances, id)

				select {
				case ch <- NilFielder(id):
				case <-ctx.Done():
					return
				}
			}

			for id, instance := range set {
				instances[id] = &Instance{*instance}
			}

			sort.Slice(ptrs, func(i, j int) bool {
				return ptrs[i].ID < ptrs[j].ID
			})

			for _, ptr := range ptrs {
				select {
				case ch <- &Instance{*ptr}:
				case <-ctx.Done():
					return
				}
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(500*time.Millisecond):
			}
		}
	}()

	selections := make(chan string)

	ctx, cancel := context.WithCancel(ctx)
	go menu(ctx, ch, selections)
	var ids []string
	for id := range selections {
		ids = append(ids, id)
		cancel()
	}

	var r []Instance
	for _, id := range ids {
		if instance, ok := instances[id]; ok {
			r = append(r, *instance)
		}
	}

	return r, fetch
}

func main() {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = os.Getenv("HOME")
	}
	secrets := fmt.Sprintf("%s/%s", config, "lambdactl")
	buf, err := os.ReadFile(secrets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error - failed to read token from '%s': %v\n", secrets, err)
		os.Exit(1)
	}
	token := string(bytes.TrimSpace(buf))

	c, err := api.NewClient(&http.Client{}, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to make API Client: %s\n", err.Error())
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	ctx, menu := MenuClosure(ctx, os.Stderr)

	handle := func(verb string) {
		switch verb {
		case "c", "create":
			err = PromptCreateInstance(ctx, c, menu)
		case "i", "instances":
			_, err = PromptInstances(ctx, c, menu)
		case "t", "terminate":
			err = PromptTerminate(ctx, c, menu)
		case "s", "ssh":
			var cmd *exec.Cmd
			cmd, err = PromptSSH(ctx, c, menu)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error terminating: %+v\n", err)
				return
			}
			cancel()
			<-ctx.Done()
			fmt.Fprintf(os.Stderr, "connecting...\n")
			err = cmd.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%+v", err)
			}
			return
		default:
			fmt.Fprintf(os.Stderr, "uh '%s'\n", verb)
			return
		}
	}

	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			handle(arg)
		}
	} else {
		for {
			verb := MenuBasic(ctx, []string{"create", "instances", "ssh", "terminate"}, menu)
			if verb == "" {
				break
			}
			handle(verb)
		}
	}

	cancel()
	<-ctx.Done()

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	}


}
