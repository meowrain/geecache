#!/bin/bash
protoc --go_out=. --go-grpc_out=. geecache/geecachepb/geecachepb.proto