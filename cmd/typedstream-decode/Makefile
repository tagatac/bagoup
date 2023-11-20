build: typedstream-decode

typedstream-decode: main.o decode.o
	swiftc -v -o $@ $^

main.o: main.swift decode.h
	swiftc -v -c -import-objc-header decode.h $< -framework Foundation

decode.o: decode.m
	clang -v -c $< -fobjc-arc -fmodules

clean:
	rm -vf typedstream-decode main.o decode.o
