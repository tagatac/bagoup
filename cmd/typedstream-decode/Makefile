mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir := $(dir $(mkfile_path))

build: typedstream-decode

typedstream-decode: main.o decode.o
	clang -o $@ decode.o $< -framework Foundation

main.o: main.swift decode.h typedstream-parser-Bridging-Header.h
	swiftc -c -import-objc-header typedstream-parser-Bridging-Header.h $<

decode.o: decode.m decode.h typedstream-parser-Bridging-Header.h
	clang -c $< -fobjc-arc -fmodules -I $(mkfile_dir)

clean:
	rm -vf typedstream-decode main.o decode.o

