#!/bin/bash

#CC=gcc
#CGO_CFLAGS="-I /usr/include/bpf"
#CGO_LDFLAGS="/usr/lib64/libbpf.a"
#go CC=${CC} CGO_CFLAGS=${CGO_CFLAGS} CGO_LDFLAGS=${CGO_LDFLAGS} test -run $1 -test.v
CC=gcc CGO_CFLAGS="-I /usr/include/bpf" CGO_LDFLAGS="/usr/lib64/libbpf.a" go test -run $1 -test.v
