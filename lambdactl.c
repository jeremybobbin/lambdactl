#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <termios.h>
#include <unistd.h>

static struct termios term[2];
static struct winsize ws;
char *key;

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

	printf("pipe %d %d\n", pp[0], pp[1]);

	*fd = pp[0];

	fprintf(stderr, "forking!\n");
	switch ((n = fork())) {
	case -1:
		close(pp[0]);
		close(pp[1]);
		return -1;
	case 0:
		dup2(pp[1], 1);
		close(pp[0]);
		return execlp("curl", "curl", "-s", "https://cloud.lambda.ai/api/v1/instances", "-u", key, NULL);
	default:
		close(pp[1]);
		return n;
	}

	return 0;

}

int fetch_ssh_keys(int *fd) {
	int n, pp[2];

	if (fd == NULL) {
		errno = EINVAL;
	}

	if (pipe(pp) == -1) {
		return -1;
	}

	printf("pipe %d %d\n", pp[0], pp[1]);

	*fd = pp[0];

	fprintf(stderr, "forking!\n");
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

int main(/*int argc, char *argv[]*/) {
	char buf[4096];
	int tty, fd, n;

	if ((key = getenv("LAMBDA_API_KEY")) == NULL || strlen(key) == 0) {
		fprintf(stderr, "LAMBDA_API_KEY missing\n");
		return 1;
	}

	fprintf(stderr, "KEY %s\n", key);

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

	if (ioctl(tty, TIOCGWINSZ, &ws) < 0) {
		perror("failed to get tty size");
		return 1;
	}

	fprintf(stderr, "got width & height: %d, %d\n", ws.ws_col, ws.ws_row);

	if (tcsetattr(tty, TCSANOW, &term[0]) == -1) {
		perror("failed to teardown tty");
		return 1;
	}

	/*
	n = fetch_instances(&fd);
	printf("fetch instances %d %d\n", n, fd);
	copy(1, fd);
	*/

	n = fetch_ssh_keys(&fd);
	printf("fetch instances %d %d\n", n, fd);
	copy(1, fd);

	//match_ssh_key("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC2xqx6t8MBfheMevVi/n4XlA4T6hJgmrqgpH4W2epmc4tGPoE2EQjmk5QnXLc1jsYoxreHaVFCFIiz5y8XkxgPJxf5hiq4s42/g1xA3w/P4MVg/frDpa4rtSalXHXWJ9Piymcykeyeb8hlhcCU5RVqy1ftCjNHycKLWvGpdPDnU7Q/GVhR5qbDLwmDxwb0U85C9LGolnY6uiYLR4CfBNsDaZiRN1Re7IIzWLmU6MGNpewEO680IqoOtQyikI/NEyWdKqQpO4TAyNl994obBu8ucsq9BahPyCzHnCf37EVUB8Lz632ZRLp6RkG0KdmzFF4gJ+ANLwoE0zWKaBoclSKgEsxzMwLBO/AJ0HhsCfglFWDGr/kGxyrg9T1ERzYEL3882aHVnQMJ8A3jSxadVev9xUEBTRz4cCQVMjWieOz1qUj3sZHMMoxK80VgBEOxODsZ2ikIpDioamlzRSOhn0J9zZ7eGUkKlsJxbTPQtkxguFiJl9mg4Ym6P7mhZv9/HLc= jer@Amphibian\n");
	return 0;
}


