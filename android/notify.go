//go:build android

package android

/*
#include <stdlib.h>
#include "activity.h"
*/
import "C"

import "unsafe"

// Channel is the channel the program's notifications go on: Android 8 and later list it, by its
// label, in the application's settings, where the person can turn it off. Set it before the first
// notification; the label follows the person's language best when the program sets it.
var Channel = struct{ ID, Label string }{ID: "messages", Label: "Messages"}

// Notify shows a notification in the system's shade, as other applications' are, even while the
// program is in the background: a title and a line, and touching it opens the program. The same
// id replaces the notification shown with it before. It returns false when it could not, most
// often because the person has not allowed notifications (NotificationsAllowed). Safe from any
// goroutine.
func Notify(id int, title, text string) bool {
	if !NotificationsAllowed() {
		return false
	}
	strings := []*C.char{C.CString(Channel.ID), C.CString(Channel.Label), C.CString(title), C.CString(text)}
	defer func() {
		for _, text := range strings {
			C.free(unsafe.Pointer(text))
		}
	}()
	return C.post_notification(C.int(id), strings[0], strings[1], strings[2], strings[3]) != 0
}

// NotificationsAllowed tells whether the program may show notifications: always before Android 13,
// and on Android 13 and later once the person allowed them (AskForNotifications).
func NotificationsAllowed() bool { return C.notifications_allowed() != 0 }

// AskForNotifications shows Android's question to allow the program's notifications, on Android 13
// and later; the package's manifest must name android.permission.POST_NOTIFICATIONS. Ask when the
// person does something that needs them, not at start. The answer is read with
// NotificationsAllowed afterwards.
func AskForNotifications() { C.ask_for_notifications() }
