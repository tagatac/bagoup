// Copyright (C) 2023  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.
//
//  decode.h
//  typedstream-decode
//

#ifndef decode_h
#define decode_h

#import <Foundation/Foundation.h>

@interface Decode : NSObject

- (void) decode:(NSString *) filename;
- (void) decodeStdin;

@end

#endif /* decode_h */
