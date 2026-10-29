// Android's document chooser, opened for Go. Android hands back what a person chose to the
// activity's onActivityResult, which NativeActivity does not pass on to native code; a fragment
// with no view gets its own onActivityResult, so this one joins the activity for one choice:
// it opens the chooser, hands Go each file the person chose (chosen), then the end (done), and
// leaves. Go hears the end exactly once for every chooser that open accepted, whatever happens.
//
// android.app.Fragment is deprecated in favour of the support libraries' fragment, a dependency
// go-renderer does not have; the platform's one still works on every Android an app runs on.
package io.github.akrck02.gorenderer;

import android.app.Activity;
import android.app.Fragment;
import android.content.ActivityNotFoundException;
import android.content.ClipData;
import android.content.ContentResolver;
import android.content.Intent;
import android.database.Cursor;
import android.net.Uri;
import android.os.Bundle;
import android.provider.OpenableColumns;
import android.util.Log;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.List;

@SuppressWarnings("deprecation")
public final class FileChooser extends Fragment {
	// Calls into Go, registered by files.c: one chosen file, with its name, its size (-1 when
	// unknown) and its bytes, or null where they were not read; then the end of the choice.
	static native void chosen(String name, long size, byte[] content);
	static native void done();

	private static final String TAG = "io.github.akrck02.gorenderer.FileChooser";
	private static final String LARGEST = "largest"; // the argument: the largest file Go takes, in bytes
	private static final String OPEN = "open"; // kept across a rebuilt activity: the chooser is already open
	private static final int CHOOSING = 1; // the request code of the chooser, the only one this fragment makes
	private static final int READ_BYTES = 64 * 1024; // how much is read at a time
	private static final long UNKNOWN = -1; // a size the other app does not tell

	private boolean answered; // Go was told, or is being told, the end of this choice

	// open adds a chooser fragment to the activity, on its main thread, as fragments must be.
	// Go hears back through done in every case, also when the chooser could not open.
	public static boolean open(final Activity activity, final long largest) {
		activity.runOnUiThread(() -> {
			if (activity.isFinishing() || activity.isDestroyed() || activity.getFragmentManager().isStateSaved()) {
				answerNothing(); // the fragment would be dropped without a word, and Go would wait for ever
				return;
			}
			FileChooser chooser = new FileChooser();
			Bundle arguments = new Bundle();
			arguments.putLong(LARGEST, largest);
			chooser.setArguments(arguments);
			try {
				activity.getFragmentManager().beginTransaction().add(chooser, TAG).commitAllowingStateLoss();
			} catch (RuntimeException failed) {
				Log.w("Go", "the file chooser could not join the activity", failed);
				chooser.answered = true;
				answerNothing();
			}
		});
		return true;
	}

	// answerNothing tells Go the choice ended empty, from a thread of its own: Go's callback never
	// runs on Android's main thread.
	private static void answerNothing() {
		new Thread(FileChooser::done, "file chooser").start();
	}

	@Override
	public void onCreate(Bundle saved) {
		super.onCreate(saved);
		if (saved != null && saved.getBoolean(OPEN)) return; // a rebuilt activity: the chooser is open
		Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT);
		intent.addCategory(Intent.CATEGORY_OPENABLE);
		intent.setType("*/*");
		intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true);
		try {
			startActivityForResult(intent, CHOOSING);
		} catch (ActivityNotFoundException missing) {
			Log.w("Go", "this phone has no document chooser", missing);
			finish(new ArrayList<>());
		}
	}

	@Override
	public void onSaveInstanceState(Bundle state) {
		super.onSaveInstanceState(state);
		state.putBoolean(OPEN, true);
	}

	@Override
	public void onActivityResult(int request, int result, Intent data) {
		if (request != CHOOSING) return;
		finish(result == Activity.RESULT_OK ? urisOf(data) : new ArrayList<>());
	}

	// onDestroy tells Go the choice ended empty when the activity goes for good before the chooser
	// answered: no rebuilt activity will get the answer.
	@Override
	public void onDestroy() {
		super.onDestroy();
		if (!answered && getActivity() != null && getActivity().isFinishing()) {
			answered = true;
			answerNothing();
		}
	}

	// finish reads the chosen files away from the main thread, so a big one does not freeze the
	// app, tells Go, and takes the fragment off the activity.
	private void finish(final List<Uri> uris) {
		answered = true;
		final ContentResolver resolver = getActivity().getContentResolver();
		final long largest = getArguments().getLong(LARGEST);
		new Thread(() -> tell(resolver, uris, largest), "file chooser").start();
		getFragmentManager().beginTransaction().remove(this).commitAllowingStateLoss();
	}

	// urisOf is every document the person chose: several come in the clip data, one in the data.
	private static List<Uri> urisOf(Intent data) {
		List<Uri> uris = new ArrayList<>();
		if (data == null) return uris;
		ClipData several = data.getClipData();
		if (several == null) {
			if (data.getData() != null) uris.add(data.getData());
			return uris;
		}
		for (int index = 0; index < several.getItemCount(); index++) {
			uris.add(several.getItemAt(index).getUri());
		}
		return uris;
	}

	// tell hands Go each document as soon as it is read, so only one is held at a time, then the end.
	private static void tell(ContentResolver resolver, List<Uri> uris, long largest) {
		for (Uri uri : uris) {
			String[] name = {"file"};
			long[] size = {UNKNOWN};
			describe(resolver, uri, name, size);
			byte[] content = size[0] <= largest ? read(resolver, uri, size[0], largest) : null;
			if (content != null && content.length > largest) {
				content = null; // longer than the other app told: too big, of a size not known
				if (size[0] <= largest) size[0] = UNKNOWN;
			} else if (content != null) {
				size[0] = content.length;
			} else if (size[0] == UNKNOWN) {
				size[0] = 0; // tried, and it could not be read
			}
			chosen(name[0], size[0], content);
		}
		done();
	}

	// describe fills in a document's name and size, as the app that holds it tells them; the size
	// stays UNKNOWN when it does not know.
	private static void describe(ContentResolver resolver, Uri uri, String[] name, long[] size) {
		String[] columns = {OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE};
		try (Cursor cursor = resolver.query(uri, columns, null, null, null)) {
			if (cursor == null || !cursor.moveToFirst()) return;
			if (!cursor.isNull(0)) name[0] = cursor.getString(0);
			if (!cursor.isNull(1)) size[0] = cursor.getLong(1);
		} catch (RuntimeException failed) {
			Log.w("Go", "could not ask for a chosen file's name", failed);
		}
	}

	// read is a document's bytes, or null when it could not be read. It stops one byte past largest:
	// an app may tell a size smaller than what it gives, and a longer answer is only too big.
	private static byte[] read(ContentResolver resolver, Uri uri, long told, long largest) {
		try (InputStream input = resolver.openInputStream(uri)) {
			if (input == null) return null;
			ByteArrayOutputStream bytes = new ByteArrayOutputStream((int) Math.max(told + 1, READ_BYTES));
			byte[] buffer = new byte[READ_BYTES];
			for (int count = input.read(buffer); count >= 0; count = input.read(buffer)) {
				bytes.write(buffer, 0, count);
				if (bytes.size() > largest) break;
			}
			return bytes.toByteArray();
		} catch (IOException | RuntimeException | OutOfMemoryError failed) {
			Log.w("Go", "could not read a chosen file", failed);
			return null;
		}
	}
}
