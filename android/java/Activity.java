package io.github.akrck02.gorenderer;

import android.app.NativeActivity;
import android.content.Intent;

// Activity is Android's NativeActivity, which runs the program's library, keeping the intent that
// brings it back to the front: a touched notification's (android.OpenedNotification reads its
// target). NativeActivity hands no new intent to the library, and getIntent goes on returning the
// one that started it.
public class Activity extends NativeActivity {
	@Override
	protected void onNewIntent(Intent intent) {
		super.onNewIntent(intent);
		setIntent(intent);
	}
}
