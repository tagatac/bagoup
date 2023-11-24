// Copyright (C) 2023  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.
//
//  decode.m
//  typedstream-decode
//

#import "decode.h"

@implementation Decode

- (void) decode:(NSString *) filename {
    NSUnarchiver *typedStreamUnarchiver = [[NSUnarchiver alloc] initForReadingWithData:[NSData dataWithContentsOfFile:filename]];
    id object = [typedStreamUnarchiver decodeObject];
    printf("%s\n", [[NSString stringWithFormat:@"%@", object] UTF8String]);
}

- (void) decodeStdin {
    NSFileHandle *input = [NSFileHandle fileHandleWithStandardInput];
    NSData *inputData = [NSData dataWithData:[input readDataToEndOfFile]];
    NSUnarchiver *typedStreamUnarchiver = [[NSUnarchiver alloc] initForReadingWithData:inputData];
    id object = [typedStreamUnarchiver decodeObject];
    printf("%s\n", [[NSString stringWithFormat:@"%@", object] UTF8String]);
}

@end
