package windowloop

// #include <pthread.h>
import "C"

// OnMainThread tells whether the calling thread is the process's main thread, the only one macOS
// delivers window events on.
func OnMainThread() bool { return C.pthread_main_np() != 0 }
