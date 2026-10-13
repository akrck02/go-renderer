//go:build android

#include "textflag.h"

// func Call(code uintptr)
TEXT ·Call(SB), $0-8
	MOVD code+0(FP), R0
	BL (R0)
	RET
