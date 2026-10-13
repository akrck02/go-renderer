//go:build android

#include "textflag.h"

// func Call(code uintptr)
TEXT ·Call(SB), $0-8
	MOVQ code+0(FP), AX
	CALL AX
	RET
