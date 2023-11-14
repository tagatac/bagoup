SWIFT_LIB_DIR="$(shell xcode-select -p)/Toolchains/XcodeDefault.xctoolchain/usr/lib/swift"

build: typedstream-decode

typedstream-decode: main.o decode.o
	echo $(SWIFT_LIB_DIR)
	clang -o $@ $^ -I$(SWIFT_LIB_DIR)

main.o: main.swift decode.h
	swiftc -c -import-objc-header decode.h $< -framework Foundation

decode.o: decode.m
	clang -c $< -fobjc-arc -fmodules

clean:
	rm -vf typedstream-decode main.o decode.o
