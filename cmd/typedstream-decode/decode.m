// Copyright (C) 2023  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.
//
//  decode.m
//  typedstream-decode
//

#import <Foundation/Foundation.h>

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        NSFileHandle *input = [NSFileHandle fileHandleWithStandardInput];
        NSData *inputData = [NSData dataWithData:[input readDataToEndOfFile]];
        NSUnarchiver *typedStreamUnarchiver = [[NSUnarchiver alloc] initForReadingWithData:inputData];
        id object = [typedStreamUnarchiver decodeObject];
        printf("%s\n", [[NSString stringWithFormat:@"%@", object] UTF8String]);
    }
    return 0;
}
