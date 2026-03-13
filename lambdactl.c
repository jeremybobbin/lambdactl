#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/select.h>
#include <sys/wait.h>
#include <termios.h>
#include <unistd.h>
#include <signal.h>

#define PADDING 2
#define CONTROL(ch)   (ch ^ 0x40)

#define MIN(a,b)      ((a) < (b) ? (a) : (b))
#define MAX(a,b)      ((a) > (b) ? (a) : (b))
#define LENGTH(x)  ((int)(sizeof (x) / sizeof *(x)))

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

/* menu
 * provides a readable outfd         - read selected options here
 * provides a writable optionfd pipe - write options here
 * provides a writable ctlfd         - close this to stop the menu
 * returns -1 if pipes or fork fail
 * subprocess exits on error
 */
int menu(int *outfd, int *optionfd) {
	int item_pipe[2], out_pipe[2];
	int n;

	if (optionfd == NULL || outfd == NULL) {
		errno = EINVAL;
		return -1;
	}

	if (pipe(out_pipe) == -1) {
		return -1;
	}

	if (pipe(item_pipe) == -1) {
		return -1;
	}

	switch (n = fork()) {
	case -1:
		perror("fork");
		exit(1);
		break;
	case 0:
		break;
	default:
		// back to main process
		// close the write-end of the out-pipe
		// close the read-ends of the item-pipe & ctl-pipe
		close(out_pipe[1]);
		close(item_pipe[0]);
		*outfd = out_pipe[0];
		*optionfd = item_pipe[1];
		return n;
	}

	close(out_pipe[0]);
	close(item_pipe[1]);

	dup2(item_pipe[0], 0);
	dup2(out_pipe[1], 1);

	return execl("./menu", "./menu", NULL);
}

enum {
	NONE = 0,
	SELECT_INSTANCE_TYPE,
	INSTANCES,
	SSH,
	TERMINATE,
	SELECT_SSH_KEY,
};

int main(/*int argc, char *argv[]*/) {
	char buf[4096];
	char *instance_type = NULL, *ssh_key = NULL;
	int i, n, pid, state = NONE, status;

	union {
		char *s;
	} args;

	for (i = 3; i < 128; i++) {
		if (close(i) == 0) {
			//printf("closed %d\n", i);
		}
	}

	if ((key = getenv("LAMBDA_API_KEY")) == NULL || strlen(key) == 0) {
		fprintf(stderr, "LAMBDA_API_KEY missing\n");
		return 1;
	}

	const char *items[] = {
		"create\n",
		"instances\n",
		"ssh\n",
		"terminate\n"
	};

	int optionfd = -1, outfd = -1;

	for (n = 0;;) {
		if ((pid = menu(&outfd, &optionfd)) == -1) {
			perror("menu");
		}

		switch (state) {
		case NONE:
			for (i = 0; i < LENGTH(items); i++) {
				n = write(optionfd, items[i], strlen(items[i]));
				if (n == -1) {
					perror("main write to menu");
					exit(1);
				}
			}
			break;
		case SELECT_INSTANCE_TYPE:
			switch ((n = fork())) {
			case -1:
				perror("fork");
				return 1;
			case 0:
				dup2(optionfd, 1);
				execl("bin/create/instance-types", "bin/create/instance-types", NULL);
				perror("exec bin/create/instance-types");
				exit(1);
			}
			break;
		case SSH:
			switch ((n = fork())) {
			case -1:
				perror("fork");
				return 1;
			case 0:
				dup2(optionfd, 1);
				execl("bin/ssh/instances", "bin/ssh/instances", NULL);
				perror("exec bin/ssh/instances");
				exit(1);
			}
			break;
		case INSTANCES: case TERMINATE:
			switch ((n = fork())) {
			case -1:
				perror("fork");
				return 1;
			case 0:
				dup2(optionfd, 1);
				execl("bin/instances", "bin/instances", NULL);
				perror("exec bin/instances");
				exit(1);
			}
			break;
		case SELECT_SSH_KEY:
			switch ((n = fork())) {
			case -1:
				perror("fork");
				return 1;
			case 0:
				dup2(optionfd, 1);
				execl("bin/create/list-ssh-keys", "bin/create/list-ssh-keys", NULL);
				perror("exec bin/create/list-ssh-keys");
				exit(1);
			}
			break;
		}


		close(optionfd);
		n = read(outfd, buf, sizeof(buf)-1);
		buf[n] = '\0';
		if (n > 0 && buf[n-1] == '\n') {
			buf[n-1] = '\0'; // trim trailing newline
		}
		close(outfd);
		kill(pid, SIGINT);

		switch (state) {
		case NONE:
			if (n == 0) {
				// escape presed
				return 0;
			} else if (strstr(buf, "create")) {
				state = SELECT_INSTANCE_TYPE;
			} else if (strstr(buf, "instances")) {
				state = INSTANCES;
			} else if (strstr(buf, "ssh")) {
				state = SSH;
			} else if (strstr(buf, "terminate")) {
				state = TERMINATE;
			} else {
				fprintf(stderr, "failed to match any options in start menu\n");
				state = NONE;
			}
			break;
		case INSTANCES:
			if (n == 0) {
				// escape presed
				state = NONE;
				break;
			}
			break;
		case SSH:
			if (n == 0) {
				// escape presed
				state = NONE;
				break;
			}
			switch ((n = fork())) {
			case -1:
				perror("fork");
				return 1;
			case 0:
				dup2(optionfd, 1);
				fprintf(stderr, "sshing %s\n", buf);
				execlp("ssh", "ssh", buf, NULL);
				perror("exec ssh");
				exit(1);
			}

			waitpid(n, &status, 0);
			break;
		case TERMINATE:
			if (n == 0) {
				// escape presed
				state = NONE;
				break;
			}
			switch ((pid = fork())) {
			case -1:
				perror("fork");
				return 1;
			case 0:
				execl("bin/terminate", "bin/terminate", buf, NULL);
				perror("exec bin/terminate");
				return 1;
			}
			break;
		case SELECT_INSTANCE_TYPE:
			if (n == 0) {
				// escape presed
				state = NONE;
				break;
			}
			instance_type = strdup(buf);
			state = SELECT_SSH_KEY;
			break;
		case SELECT_SSH_KEY:
			if (n == 0) {
				// escape presed
				state = SELECT_INSTANCE_TYPE;
				break;
			}
			ssh_key = strdup(buf);
			switch ((pid = fork())) {
			case -1:
				perror("fork");
				return 1;
			case 0:
				execl("bin/create/launch", "bin/create/launch", instance_type, buf, NULL);
				perror("exec bin/create/launch");
				return 1;
			}


			for (;;) {
				n = waitpid(pid, &status, 0);
				if (n == -1) {
					if (errno == EINTR) {
						continue;
					}
					status = -1;
					break;
				}
				break;
			}
			state = INSTANCES;
			break;
		}
	}

	return 0;
}
