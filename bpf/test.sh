#!/bin/bash

./buildbpf.sh userFuncCount
./buildbpf.sh userFuncExecTime

./buildgo.sh bpf_test

sudo ./bpf_test $1 $2 $3
