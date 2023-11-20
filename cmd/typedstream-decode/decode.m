//
//  decode.m
//  typedstream-decode
//
//  Created by David Tagatac on 11/12/23.
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
