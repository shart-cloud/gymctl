# Jerry's "it builds on my laptop" image.
#
# One giant stage built on full Ubuntu: it apt-installs the entire Go toolchain,
# a C compiler, git and curl — then SHIPS all of it to production. The container
# also runs as root (there is no USER instruction).
#
# The binary is ~7MB. Everything else in this ~1GB image is attack surface you
# never needed at runtime: a shell, a package manager, compilers, headers.
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y \
        golang-go \
        build-essential \
        git \
        curl \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /usr/local/bin/app .

EXPOSE 8080
# No USER — this runs as root.
CMD ["/usr/local/bin/app"]
