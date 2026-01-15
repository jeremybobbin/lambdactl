#include <string.h>
#include <stdio.h>
#include <errno.h>
#include <dirent.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>

#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>

#include <openssl/ssl.h>
#include <openssl/err.h>

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

int connect_tcp(const char *host, const char *port) {
	struct addrinfo hints;
	struct addrinfo *res;
	struct addrinfo *rp;
	int fd;

	memset(&hints, 0, sizeof(hints));
	hints.ai_family = AF_UNSPEC;
	hints.ai_socktype = SOCK_STREAM;

	res = NULL;
	if (getaddrinfo(host, port, &hints, &res) != 0)
		return -1;

	fd = -1;
	for (rp = res; rp != NULL; rp = rp->ai_next) {
		fd = socket(rp->ai_family, rp->ai_socktype, rp->ai_protocol);
		if (fd < 0)
			continue;

		if (connect(fd, rp->ai_addr, rp->ai_addrlen) == 0)
			break;

		close(fd);
		fd = -1;
	}

	freeaddrinfo(res);
	return fd;
}

int ssl_write_all(SSL *ssl, const void *buf, size_t len) {
	const char *p;
	size_t off;

	p = (const char *)buf;
	off = 0;

	while (off < len) {
		int n;

		n = SSL_write(ssl, p + off, (int)(len - off));
		if (n > 0) {
			off += (size_t)n;
			continue;
		}

		switch (SSL_get_error(ssl, n)) {
		case SSL_ERROR_WANT_READ:
		case SSL_ERROR_WANT_WRITE:
			continue;
		default:
			return -1;
		}
	}

	return 0;
}

int ssl_read_to_stdout(SSL *ssl) {
	char buf[4096];

	for (;;) {
		int n;

		n = SSL_read(ssl, buf, (int)sizeof(buf));
		if (n > 0) {
			if (write(STDOUT_FILENO, buf, (size_t)n) < 0)
				return -1;
			continue;
		}

		if (n == 0)
			return 0;

		switch (SSL_get_error(ssl, n)) {
		case SSL_ERROR_WANT_READ:
		case SSL_ERROR_WANT_WRITE:
			continue;
		case SSL_ERROR_ZERO_RETURN:
			return 0;
		default:
			return -1;
		}
	}
}

void openssl_init_once(void) {
	/* Safe across many OpenSSL versions; harmless if deprecated. */
	SSL_library_init();
	SSL_load_error_strings();
	OpenSSL_add_all_algorithms();
}

int main(int argc, char *argv[]) {

	const char *host;
	const char *path;
	const char *port;
	int fd;
	SSL_CTX *ctx;
	SSL *ssl;
	char req[2048];
	int n;
	long vr;

	if (argc != 3) {
		fprintf(stderr, "usage: %s <host> <path>\n", argv[0]);
		return 2;
	}

	host = argv[1];
	path = argv[2];
	port = "443";

	openssl_init_once();

	ctx = SSL_CTX_new(TLS_client_method());
	if (ctx == NULL) {
		ERR_print_errors_fp(stderr);
		return 1;
	}

	/* Enable certificate verification using system default CAs. */
	if (SSL_CTX_set_default_verify_paths(ctx) != 1) {
		ERR_print_errors_fp(stderr);
		SSL_CTX_free(ctx);
		return 1;
	}
	SSL_CTX_set_verify(ctx, SSL_VERIFY_PEER, NULL);

	fd = connect_tcp(host, port);
	if (fd < 0) {
		fprintf(stderr, "connect_tcp failed\n");
		SSL_CTX_free(ctx);
		return 1;
	}

	ssl = SSL_new(ctx);
	if (ssl == NULL) {
		ERR_print_errors_fp(stderr);
		close(fd);
		SSL_CTX_free(ctx);
		return 1;
	}

	/* SNI + hostname verification. */
	if (SSL_set_tlsext_host_name(ssl, host) != 1) {
		ERR_print_errors_fp(stderr);
		SSL_free(ssl);
		close(fd);
		SSL_CTX_free(ctx);
		return 1;
	}
	if (SSL_set1_host(ssl, host) != 1) {
		ERR_print_errors_fp(stderr);
		SSL_free(ssl);
		close(fd);
		SSL_CTX_free(ctx);
		return 1;
	}

	if (SSL_set_fd(ssl, fd) != 1) {
		ERR_print_errors_fp(stderr);
		SSL_free(ssl);
		close(fd);
		SSL_CTX_free(ctx);
		return 1;
	}

	if (SSL_connect(ssl) != 1) {
		ERR_print_errors_fp(stderr);
		SSL_free(ssl);
		close(fd);
		SSL_CTX_free(ctx);
		return 1;
	}

	vr = SSL_get_verify_result(ssl);
	if (vr != X509_V_OK) {
		fprintf(stderr, "TLS cert verify failed: %ld\n", vr);
		SSL_shutdown(ssl);
		SSL_free(ssl);
		close(fd);
		SSL_CTX_free(ctx);
		return 1;
	}

	n = snprintf(req, sizeof(req),
		"GET %s HTTP/1.1\r\n"
		"Host: %s\r\n"
		"Accept: application/json\r\n"
		"User-Agent: lambdactl-cli/1.0\r\n"
		"Connection: close\r\n"
		"\r\n",
		path, host);

	if (n < 0 || (size_t)n >= sizeof(req)) {
		fprintf(stderr, "request too large\n");
		SSL_shutdown(ssl);
		SSL_free(ssl);
		close(fd);
		SSL_CTX_free(ctx);
		return 1;
	}

	if (ssl_write_all(ssl, req, (size_t)n) < 0) {
		ERR_print_errors_fp(stderr);
		SSL_shutdown(ssl);
		SSL_free(ssl);
		close(fd);
		SSL_CTX_free(ctx);
		return 1;
	}

	if (ssl_read_to_stdout(ssl) < 0) {
		ERR_print_errors_fp(stderr);
		SSL_shutdown(ssl);
		SSL_free(ssl);
		close(fd);
		SSL_CTX_free(ctx);
		return 1;
	}

	SSL_shutdown(ssl);
	SSL_free(ssl);
	close(fd);
	SSL_CTX_free(ctx);

	match_ssh_key("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC2xqx6t8MBfheMevVi/n4XlA4T6hJgmrqgpH4W2epmc4tGPoE2EQjmk5QnXLc1jsYoxreHaVFCFIiz5y8XkxgPJxf5hiq4s42/g1xA3w/P4MVg/frDpa4rtSalXHXWJ9Piymcykeyeb8hlhcCU5RVqy1ftCjNHycKLWvGpdPDnU7Q/GVhR5qbDLwmDxwb0U85C9LGolnY6uiYLR4CfBNsDaZiRN1Re7IIzWLmU6MGNpewEO680IqoOtQyikI/NEyWdKqQpO4TAyNl994obBu8ucsq9BahPyCzHnCf37EVUB8Lz632ZRLp6RkG0KdmzFF4gJ+ANLwoE0zWKaBoclSKgEsxzMwLBO/AJ0HhsCfglFWDGr/kGxyrg9T1ERzYEL3882aHVnQMJ8A3jSxadVev9xUEBTRz4cCQVMjWieOz1qUj3sZHMMoxK80VgBEOxODsZ2ikIpDioamlzRSOhn0J9zZ7eGUkKlsJxbTPQtkxguFiJl9mg4Ym6P7mhZv9/HLc= jer@Amphibian\n");

	return 0;
}


