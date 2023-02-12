#!/bin/bash

clang -g -O2 -c -target bpf -I../include -o newProcess.bpf.c newProcess.bpf.o
