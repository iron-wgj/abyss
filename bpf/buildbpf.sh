#!/bin/bash

clang -g -O2 -c -target bpf -I../include/ -o $1.bpf.o $1.bpf.c
#clang $FLAGS -o $1.bpf.o $1.bpf.c
