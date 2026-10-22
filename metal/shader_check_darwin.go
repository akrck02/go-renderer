package metal

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Metal -framework Foundation
#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#include <stdlib.h>
#include <string.h>

// shaderCheckNoDevice tells the Go side that the failure is a missing Metal device, not the source.
static const int shaderCheckNoDevice = 1;
static const int shaderCheckFailed = 2;

// checkShaderSource compiles the source and looks up each function. It returns 0 when all is well;
// otherwise a failure code and, in failure, a message the caller frees.
static int checkShaderSource(const char *source, const char **functions, int functionCount, char **failure) {
	@autoreleasepool {
		id<MTLDevice> device = MTLCreateSystemDefaultDevice();
		if (device == nil) {
			*failure = strdup("this computer has no Metal device");
			return shaderCheckNoDevice;
		}
		MTLCompileOptions *options = [MTLCompileOptions new];
		// the same safe maths the renderer compiles with, so what is checked is what runs
		if (@available(macOS 15.0, *)) {
			options.mathMode = MTLMathModeSafe;
		}
		NSError *error = nil;
		id<MTLLibrary> library = [device newLibraryWithSource:[NSString stringWithUTF8String:source] options:options error:&error];
		if (library == nil) {
			*failure = strdup([NSString stringWithFormat:@"compiling the shaders: %@", error.localizedDescription].UTF8String);
			return shaderCheckFailed;
		}
		NSMutableArray<NSString *> *missing = [NSMutableArray array];
		for (int index = 0; index < functionCount; index++) {
			NSString *name = [NSString stringWithUTF8String:functions[index]];
			if ([library newFunctionWithName:name] == nil) {
				[missing addObject:name];
			}
		}
		if (missing.count > 0) {
			*failure = strdup([NSString stringWithFormat:@"functions not in the library: %@", [missing componentsJoinedByString:@", "]].UTF8String);
			return shaderCheckFailed;
		}
		return 0;
	}
}
*/
import "C"

import (
	"unsafe"

	"github.com/akrck02/go-renderer/failure"
)

// compileShaderSource compiles Metal Shading Language at run time, as the renderer does, and checks
// that every named function is in the result. The command-line tools have no offline Metal
// compiler, so this is how the shaders are checked without Xcode. A computer without a Metal
// device gives an error wrapping ErrUnavailable.
func compileShaderSource(source string, functions []string) error {
	cSource := C.CString(source)
	defer C.free(unsafe.Pointer(cSource))
	cFunctions := make([]*C.char, len(functions))
	for index, name := range functions {
		cFunctions[index] = C.CString(name)
		defer C.free(unsafe.Pointer(cFunctions[index]))
	}
	var functionList **C.char
	if len(cFunctions) > 0 {
		functionList = &cFunctions[0]
	}
	var problem *C.char
	result := C.checkShaderSource(cSource, functionList, C.int(len(functions)), &problem)
	if result == 0 {
		return nil
	}
	defer C.free(unsafe.Pointer(problem))
	message := C.GoString(problem)
	if result == C.shaderCheckNoDevice {
		return failure.Wrap(Unavailable, message, nil)
	}
	return failure.Wrap(ShaderNotCompiled, message, nil)
}
