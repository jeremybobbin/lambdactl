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

typedef struct Pair {
	char *key;
	char *value;
} Pair;

int find(Pair *pairs, int len, char *key) {
	int i;

	for (i = 0; i < len; i++) {
		if (strcmp(pairs[i].key, key) == 0) {
			return i;
		}
	}

	return -1;
}

// read tsv into options array
int read_tsv(int fd, Pair **options, int *len) {
	int m;
	static int i, j, k, n, off;
	static char buf[2048]; // TODO - set to 10 & fix
	char *option, *str;


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

	// if there's a tab somewhere in the line, set the first field as the key & the rest as the value
	for (i = 0, j = 0; i < n; i++) {
		if (buf[i] == '\n') {
			if ((*options = realloc(*options, sizeof(**options)*(*len+1))) == NULL) {
				perror("malloc options");
				break;
			}

			if ((str = memchr(&buf[j], '\t', i-j))) {
				off = str - &buf[j];
				if ((option = malloc(off+1)) == NULL) {
					perror("malloc option");
					exit(1);
				}

				memcpy(option, &buf[j], off+1);
				option[off] = '\0';
				(*options)[*len].key = option;

				off += 1;

				if ((option = malloc(i-j-off+1)) == NULL) {
					perror("malloc option");
					exit(1);
				}
				memcpy(option, &buf[j+off], i-j-off);
				option[i-j-off] = '\0';
				(*options)[*len].value = option;
			} else {
				if ((option = malloc(i-j+1)) == NULL) {
					perror("malloc option");
					exit(1);
				}
				memcpy(option, &buf[j], i-j);
				option[i-j] = '\0';

				if ((*options = realloc(*options, sizeof(**options)*(*len+1))) == NULL) {
					perror("malloc options");
					break;
				}
				(*options)[*len].key = option;
				(*options)[*len].value = option;
			}

			if ((k = find(*options, *len, (*options)[*len].key)) != -1) {
				(*options)[k].value = (*options)[*len].value;
			} else {
				(*len)++;
			}

			j = i+1;
		}
	}

	memmove(buf, &buf[i], n-i);
	n -= i;
	i = 0;

	return 1;
}

// items - array of tsv
char **stretch(Pair *items, int len) {
	int i, j, n, m, max = 0, pad, remaining, *widths;
	char **columns, **rows, *row, *str;

	// for every row, count the columns, to learn the maximum number of columns required
	for (i = 0; i < len; i++) {
		m = 1;
		str = items[i].value;
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
		str = items[i].value;

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
		str = items[i].value;
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

int main(/*int argc, char *argv[]*/) {
	int i, n, tty, sel = 0, offset = 0, len = 0;
	char buf[2048];
	Pair *options = NULL;
	fd_set rs, ws;
	offset = 0;

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

	// default colors
	write(tty, "\x1b[0m", sizeof("\x1b[0m")-1);

	for (;;) {
		FD_ZERO(&rs);
		FD_ZERO(&ws);

		FD_SET(0, &rs);
		FD_SET(tty, &rs);

		if ((n = select(5, &rs, &ws, NULL, NULL)) == -1) {
			perror("select");
			exit(1);
		}

		if (FD_ISSET(0, &rs)) {
			if (read_tsv(0, &options, &len) == 0) {
				// stdin is closed
				break;
			}
			fflush(stdout);
		}

		char **r = stretch(options, len);

		if (FD_ISSET(tty, &rs)) {
			if ((n = read(tty, buf, sizeof(buf))) == -1) {
				perror("read stdin");
				break;
			}

			switch (buf[0]) {
			case 0x40 ^ 'J': case 0x40 ^ 'N':
				sel++;
				sel = MIN(sel, len-1);
				if (sel >= offset+MIN(len, win.ws_row)) {
					offset++;
				}
				offset = MIN(offset, len-MIN(len, win.ws_row));
				offset = MAX(offset, 0);
				break;

			case 0x40 ^ 'K': case 0x40 ^ 'P':
				if (sel-1 <= 0) {
					sel = 0;
					break;
				}

				sel--;

				if (offset <= 0) {
					break;
				}
				if (sel <= offset) {
					offset--;
				}
				break;

			case '\r':
				write(1, r[sel], strlen(r[sel]));
				write(1, "\n", 1);
			}
		}

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

		if (MIN(len, win.ws_row)) {
			// cursor up n-times, cursor to column n
			fprintf(stderr, "\x1b[%dF\x1b[%dG", MIN(len, win.ws_row), len);
		}

	}
	// erase to end of screen
	fprintf(stderr, "\x1b[G\x1b[J");

	exit(0);

}
