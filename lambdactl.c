#include <string.h>
#include <stdio.h>
#include <errno.h>
#include <dirent.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>

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

int main(/*int argc, char *argv[]*/) {
	match_ssh_key("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC2xqx6t8MBfheMevVi/n4XlA4T6hJgmrqgpH4W2epmc4tGPoE2EQjmk5QnXLc1jsYoxreHaVFCFIiz5y8XkxgPJxf5hiq4s42/g1xA3w/P4MVg/frDpa4rtSalXHXWJ9Piymcykeyeb8hlhcCU5RVqy1ftCjNHycKLWvGpdPDnU7Q/GVhR5qbDLwmDxwb0U85C9LGolnY6uiYLR4CfBNsDaZiRN1Re7IIzWLmU6MGNpewEO680IqoOtQyikI/NEyWdKqQpO4TAyNl994obBu8ucsq9BahPyCzHnCf37EVUB8Lz632ZRLp6RkG0KdmzFF4gJ+ANLwoE0zWKaBoclSKgEsxzMwLBO/AJ0HhsCfglFWDGr/kGxyrg9T1ERzYEL3882aHVnQMJ8A3jSxadVev9xUEBTRz4cCQVMjWieOz1qUj3sZHMMoxK80VgBEOxODsZ2ikIpDioamlzRSOhn0J9zZ7eGUkKlsJxbTPQtkxguFiJl9mg4Ym6P7mhZv9/HLc= jer@Amphibian\n");

	return 0;
}


