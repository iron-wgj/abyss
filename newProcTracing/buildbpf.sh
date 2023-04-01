#!/bin/bash

clang -g -O2 -c -target bpf -I../include -o newProcess.bpf.o newProcess.bpf.c
