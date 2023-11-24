build: typedstream-decode

typedstream-decode: main.o decode.o
	swiftc -v -o $@ $^

main.o: main.swift decode.h
	swiftc -v -c -import-objc-header decode.h $<

decode.o: decode.m
	clang -v -c $<

clean:
	rm -vf typedstream-decode main.o decode.o

typedstream-decode-linux: decode.m
	clang -g -o $@ -I /usr/include/GNUstep -I /usr/local/include/GNUstep $< -l objc -l gnustep-base

typedstream-decode-objc2: decode.m
	clang -g -o $@ -I /usr/include/GNUstep -I /usr/local/include/GNUstep $< -L /usr/local/lib -l objc -l gnustep-base