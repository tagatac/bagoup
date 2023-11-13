//
//  decode.h
//  typedstream-parser
//
//  Created by David Tagatac on 11/12/23.
//

#ifndef decode_h
#define decode_h

#import <Foundation/Foundation.h>

@interface Decode : NSObject

- (void) decode:(NSString *) filename;
- (void) decodeStdin;

@end

#endif /* decode_h */
