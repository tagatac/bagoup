build: typedstream-decode

typedstream-decode: main.o decode.o
	clang -o $@ $^

main.o: main.swift decode.h
	swiftc -c -import-objc-header decode.h $<

decode.o: decode.m
	clang -c $< -fobjc-arc -fmodules

clean:
	rm -vf typedstream-decode main.o decode.o
