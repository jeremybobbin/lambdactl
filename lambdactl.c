#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/select.h>
#include <termios.h>
#include <unistd.h>

#define PADDING 2
#define CONTROL(ch)   (ch ^ 0x40)

#define MIN(a,b)      ((a) < (b) ? (a) : (b))
#define MAX(a,b)      ((a) > (b) ? (a) : (b))
#define LENGTH(x)  ((int)(sizeof (x) / sizeof *(x)))

static struct termios term[2];
static struct winsize win;
char *key;

int draw_line(int fd, char *text, int color) {
	char *buf;
	int i, j, n;

	if ((buf = malloc((win.ws_col-PADDING))) == NULL) {
		perror("malloc");
		exit(1);
	}

	for (i = 0, j = 0, n = 0; i < (win.ws_col-PADDING); i++) {
		if (n > 0) {
			n--;
			buf[i] = ' ';
		} else if (j < (int) strlen(text)) {
			switch (text[j]) {
			case '\t':
				n = 7;
				buf[i] = ' ';
				j++;
				break;
			default:
				buf[i] = text[j];
				break;
				j++;
			}
		} else {
			buf[i] = ' ';
		}
	}

	switch (color) {
	case 0: // default
		return dprintf(fd, "\n\x1b[2K %s ", buf);
	default: // reverse
		// cursor column n
		return dprintf(fd, "\n\x1b[2K\x1b[7m %s \x1b[0m", buf);
	}
}


// read tsv into options array
int read_tsv(int fd, char ***options, int *len) {
	int m;
	static int i, j, n;
	static char buf[2048]; // TODO - set to 10 & fix
	char *option;

	if (options == NULL) {
		fprintf(stderr, "read_tsv failure\n");
		exit(1);
	}

	if ((m = read(fd, &buf[i], sizeof(buf)-i)) == -1) {
		perror("read in read_tsv");
		return -1;
	}

	if (m == 0) {
		return 0;
	}

	n += m;

	for (i = 0, j = 0; i < n; i++) {
		if (buf[i] == '\n') {
			if ((option = malloc(i-j)) == NULL) {
				perror("malloc option");
				exit(1);
			}
			memcpy(option, &buf[j], i-j);
			if ((*options = realloc(*options, sizeof(**options)*(*len+1))) == NULL) {
				perror("malloc options");
				break;
			}
			(*options)[*len] = option;
			(*len)++;
			j = i+1;
		}
	}

	memmove(buf, &buf[i], n-i);
	n -= i;
	i = 0;

	return 1;
}

// items - array of tsv
char **stretch(char **items, int len) {
	int i, j, n, m, max = 0, pad, remaining, *widths;
	char **columns, **rows, *row, *str;

	// for every row, count the columns, to learn the maximum number of columns required
	for (i = 0; i < len; i++) {
		m = 1;
		str = items[i];
		//fprintf(stderr, "stretching: %s\n", items[i]);
		for (j = 0; str[j]; j++) {
			if (str[j] == '\t') {
				m++;
			}
		}

		max = MAX(m, max);
	}

	if ((columns = malloc(sizeof(*columns)*max)) == NULL) {
		return NULL;
	}

	// this is the max widths of each column
	if ((widths = malloc(sizeof(*widths)*max)) == NULL) {
		return NULL;
	}

	for (i = 0; i < len; i++) {
		str = items[i];

		j = 0;
		while (*str) {
			for (n = 0; str[n] && str[n] != '\t'; n++);
			widths[j] = MAX(widths[j], n);

			if (str[n] == '\0') {
				break;
			} 

			str += n + 1;
			j++;
		}
	}

	remaining = win.ws_col - PADDING;
	for (i = 0; i < max; i++) {
		remaining -= widths[i];
	}

	if ((rows = malloc(sizeof(*rows)*len)) == NULL) {
		return NULL;
	}

	if (max > 1) {
		pad = remaining / (max - 1);
	} else {
		pad = remaining;
	}

	for (i = 0; i < len; i++) {
		if ((rows[i] = malloc(sizeof(**rows)*win.ws_col)) == NULL) {
			return NULL;
		}

		row = rows[i];
		str = items[i];
		j = 0;
		while (str && *str) {
			for (n = 0; str[n] && str[n] != '\t'; n++);

			if (j == max-1) {
				row = row + sprintf(row, "%*.*s", widths[j], n, str);
			} else {
				row = row + sprintf(row, "%-*.*s", pad+widths[j], n, str);
			}
			j++;

			if (!str[n]) {
				break;
			}
			str += n+1;
		}
	}

	return rows;
}


int fdstrcmp(int fd, const char *s) {
	int len, i, n;
	char buf[4096];

	if (s == NULL) {
		errno = EINVAL;
		return -1;
	}

	len = strlen(s);
	for (i = 0;;) {
		if ((n = read(fd, buf, sizeof(buf))) < 0) {
			return -1;
		}

		if (n == 0) {
			return (i == len) ? 1 : 0;
		}

		if (n > len - i) {
			return 0;
		}

		if (memcmp(buf, s + i, n) != 0) {
			return 0;
		}

		i += n;

		if (i == len) {
			for (;;) {
				n = read(fd, buf, sizeof(buf));
				if (n < 0) {
					return -1;
				}
				return (n == 0) ? 1 : 0;
			}
		}
	}
}

int match_ssh_key(char *key) {
	DIR *dir;
	struct dirent *dp;
	char path[4096], *s;
	int n, fd;

	if ((s = getenv("HOME")) == NULL) {
		return -1;
	}

	sprintf(&path[0], "%s/.ssh", s);

	if ((dir = opendir(path)) == NULL) {
		perror("couldn't open directory");
		return -1;
	}

	while ((dp = readdir(dir)) != NULL) {
		if (strcmp(dp->d_name, ".") == 0 || strcmp(dp->d_name, "..") == 0) {
			continue;
		}
		if ((n = strlen(dp->d_name)) < 4) {
			continue;
		}

		if (strcmp(&dp->d_name[n-4], ".pub") != 0) {
			continue;
		}


		sprintf(&path[0], "%s/.ssh/%s", s, dp->d_name);
		if ((fd = open(path, O_RDONLY)) == -1) {
			fprintf(stderr, "failed opening %s\n", path);
		}

		if ((n = fdstrcmp(fd, key)) == 1) {
			fprintf(stderr, "matched %s\n", key);
			return 1;
		}
	}

	return closedir(dir);
}

int fetch_instances(int *fd) {
	int n, pp[2];

	if (fd == NULL) {
		errno = EINVAL;
	}

	if (pipe(pp) == -1) {
		return -1;
	}

	*fd = pp[0];

	switch ((n = fork())) {
	case -1:
		close(pp[0]);
		close(pp[1]);
		return -1;
	case 0:
		dup2(pp[1], 1);
		close(pp[0]);
		return execl("bin/instances/get", "get");
	default:
		close(pp[1]);
		return n;
	}
}

int fetch_ssh_keys(int *fd) {
	int n, pp[2];

	if (fd == NULL) {
		errno = EINVAL;
	}

	if (pipe(pp) == -1) {
		return -1;
	}

	*fd = pp[0];

	switch ((n = fork())) {
	case -1:
		close(pp[0]);
		close(pp[1]);
		return -1;
	case 0:
		dup2(pp[1], 1);
		close(pp[0]);
		return execlp("curl", "curl", "-s", "https://cloud.lambda.ai/api/v1/ssh-keys", "-u", key, NULL);
	default:
		close(pp[1]);
		return n;
	}

	return 0;

}

int copy(int a, int b) {
	int n, m;
	char buf[4096];

	for (m = 0;; m += n) {
		if ((n = read(b, buf, sizeof(buf))) <= 0) {
			break;
		}

		if ((n = write(a, buf, n)) <= 0) {
			break;
		}
	}

	return m;
}

int menu(int optionfd, int out, int ttyfd, int intrfd) {
	int i, n, sel = 0, offset, color, len = 0, maxfd = 0;
	char buf[2048], **options = NULL, *option;
	fd_set rs, ws;
	offset = 0;

	maxfd = MAX(maxfd, optionfd);
	maxfd = MAX(maxfd, out);
	maxfd = MAX(maxfd, ttyfd);
	maxfd = MAX(maxfd, intrfd);

	// default colors
	write(ttyfd, "\x1b[0m", sizeof("\x1b[0m")-1);

	for (;;) {
		FD_ZERO(&rs);
		FD_ZERO(&ws);

		//FD_SET(ttyfd, &rs);
		//FD_SET(ttyfd, &ws);
		if (optionfd >= 0) {
			FD_SET(optionfd, &rs);
		}
		//FD_SET(intrfd, &rs);

		if ((n = select(maxfd+1, &rs, &ws, NULL, NULL)) == -1) {
			perror("select");
			return 1;
		}

		if (FD_ISSET(ttyfd, &rs)) {
			if ((n = read(ttyfd, buf, sizeof(buf))) == -1) {
				perror("read stdin");
				break;
			}

			switch (buf[0]) {
			}
			break;
		}

		if (FD_ISSET(intrfd, &rs)) {
			if ((n = read(intrfd, buf, sizeof(buf))) == -1) {
				perror("read intrfd in menu");
				break;
			}

			switch (buf[0]) {
			}
			break;
		}

		if (FD_ISSET(optionfd, &rs)) {
			if (read_tsv(optionfd, &options, &len) == 0) {
				optionfd = -1;
			}
		}

		char **r = stretch((char**)options, len);

		
		for (i = 0; i < len; i++) {
			switch (i == sel) {
			case 0: // default
				fprintf(stderr, "\n\x1b[2K %s ", r[i]);
				break;
			default: // reverse
				// cursor column n
				fprintf(stderr, "\n\x1b[2K\x1b[7m %s \x1b[0m", r[i]);
				break;
			}
		}

		fprintf(stderr, "\x1b[%dF\x1b[%dG", MIN(len, win.ws_row), 1);
	}

/*
loop:
	for {
		strs := stretch(r, win.ws_col)

		for (i = 0; i < len; i++){
			color = i+offset == sel && i+offset < len;
			draw_line(display, strs[i], color);
		}

		// cursor up n-times, cursor to column n
		dprintf(ttyfd, "\x1b[%dF\x1b[%dG", MIN(len, win.ws_row), 1);


		case item, ok := <-rows:
			if !ok {
				rows = nil
				continue
			}
			id := item.String()

			if i, ok := indicies[item.String()]; ok {
				items[i] = item
			} else {
				indicies[id] = len
				items = append(items, item)
			}
			continue
		case key, ok = <-keys:
			if !ok {
				break loop
			}
		case <-winch:
			win.ws_col, win.ws_row, err = dimensions(tty)
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
			if sel < 0 || sel >= len-1 {
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
			if sel < 0 || sel >= len {
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
			sel = MIN(sel, len-1)
			if sel >= offset+MIN(len, win.ws_row) {
				offset++
			}
			offset = MIN(offset, len-MIN(len, win.ws_row))
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
			offset = Max(0, len-MIN(len, win.ws_row))
			sel = Max(len-1, 0)
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
	*/
}

int main(/*int argc, char *argv[]*/) {
	char buf[4096];
	int i, n, fd, tty;
	fd_set rs;

	if ((key = getenv("LAMBDA_API_KEY")) == NULL || strlen(key) == 0) {
		fprintf(stderr, "LAMBDA_API_KEY missing\n");
		return 1;
	}

	if ((tty = open("/dev/tty", O_RDONLY)) == -1) {
		perror("failed to open tty");
		return 1;
	}

	tcgetattr(tty, &term[0]);
	term[1] = term[0];
	term[1].c_iflag &= ~(BRKINT|PARMRK|ISTRIP|INLCR|IGNCR|ICRNL|IXON);
	term[1].c_lflag &= ~(ECHO|ECHONL|IEXTEN|ICANON); // |ICANON
	term[1].c_cflag &= ~(CSIZE|PARENB);
	term[1].c_cflag |= CS8;
	term[1].c_cc[VMIN] = 1;

	if (tcsetattr(tty, TCSANOW, &term[1]) < 0) {
		perror("failed to setup tty");
		return 1;
	}

	if (ioctl(tty, TIOCGWINSZ, &win) < 0) {
		perror("failed to get tty size");
		return 1;
	}

	n = fetch_instances(&fd);
	copy(1, fd);

	/*
	n = fetch_ssh_keys(&fd);
	printf("fetch instances %d %d\n", n, fd);
	copy(1, fd);
	*/

	const char *items[] = {
		"File\tCamel!\tABC\n",
		"Cockboy yeah yeah \tCut\tABC\n",
		"View\tZoom In\tABC\n",
		"Help\tAbout\tABC\n"
	};


	int pp[2];
	if (pipe(pp) == -1) {
		perror("main pipe");
		exit(1);
	}

	switch (n = fork()) {
	case -1:
		perror("fork");
		exit(1);
		break;
	case 0:
		close(pp[1]);
		menu(pp[0], 0, 0, 0);
		return 0;
	default:
		close(pp[0]);
		for (i = 0; i < LENGTH(items); i++) {
			n = write(pp[1], items[i], strlen(items[i]));
			if (n == -1) {
				perror("main write to menu");
				exit(1);
			}
		}
		close(pp[1]);
		break;
	}

	for (;;) {
		FD_ZERO(&rs);
		FD_SET(0, &rs);

		if ((n = select(1, &rs, NULL, NULL, NULL)) == -1) {
			perror("select");
			return 1;
		}

		if (FD_ISSET(0, &rs)) {
			if ((n = read(0, &buf[0], sizeof(buf))) == -1) {
				perror("read stdin");
				break;
			}

			switch (buf[0]) {
			case 'q': case CONTROL('D'): case CONTROL('['):
				break;
			case '\r':
				continue;
			default:
				continue;
			}
			break;
		}
	}


	if (tcsetattr(tty, TCSANOW, &term[0]) == -1) {
		perror("failed to teardown tty");
		return 1;
	}

	//match_ssh_key("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC2xqx6t8MBfheMevVi/n4XlA4T6hJgmrqgpH4W2epmc4tGPoE2EQjmk5QnXLc1jsYoxreHaVFCFIiz5y8XkxgPJxf5hiq4s42/g1xA3w/P4MVg/frDpa4rtSalXHXWJ9Piymcykeyeb8hlhcCU5RVqy1ftCjNHycKLWvGpdPDnU7Q/GVhR5qbDLwmDxwb0U85C9LGolnY6uiYLR4CfBNsDaZiRN1Re7IIzWLmU6MGNpewEO680IqoOtQyikI/NEyWdKqQpO4TAyNl994obBu8ucsq9BahPyCzHnCf37EVUB8Lz632ZRLp6RkG0KdmzFF4gJ+ANLwoE0zWKaBoclSKgEsxzMwLBO/AJ0HhsCfglFWDGr/kGxyrg9T1ERzYEL3882aHVnQMJ8A3jSxadVev9xUEBTRz4cCQVMjWieOz1qUj3sZHMMoxK80VgBEOxODsZ2ikIpDioamlzRSOhn0J9zZ7eGUkKlsJxbTPQtkxguFiJl9mg4Ym6P7mhZv9/HLc= jer@Amphibian\n");
	return 0;
}


