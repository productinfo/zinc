#!/bin/bash

rm -fR pkg/core/data
rm -fR test/data

export FIRST_ADMIN_USER="admin"
export FIRST_ADMIN_PASSWORD="Complexpass#123" 

if [[ $1 == "bench" ]]; then
    go test -v -test.run=NONE -test.bench=Bulk -benchmem ./test/ -cpuprofile=./tmp/cpu.pprof -memprofile=./tmp/mem.pprof
    # go tool pprof -http=:9999 ./tmp/mem.pprof
else
    go test -v ./...
fi
