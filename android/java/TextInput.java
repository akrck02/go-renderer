// The on-screen keyboard's side of a text field. A keyboard edits what it types through the focused
// view's InputConnection: it commits words, keeps the word it is still making (composing: a
// prediction, a Chinese or Japanese reading, an emoji), and deletes around the caret. NativeActivity's
// view offers none, so the keyboard falls back to sending keys one by one, and what has no key (an
// emoji, a suggestion) is lost. This view stands in for the field Go draws: it holds a copy of the
// field's text, lets the keyboard edit it, and tells Go the whole text, the selection and the
// composing word after each edit (edited). Go sends its own changes back (set): a key from a
// hardware keyboard, a touch moving the caret, a field cutting what does not fit.
//
// The two sides edit at once, so each keyboard edit is numbered: Go says which edit its text comes
// after, and a text from before the latest edit is dropped, as the keyboard has typed since.
package io.github.akrck02.gorenderer;

import android.app.Activity;
import android.content.Context;
import android.text.Editable;
import android.text.InputType;
import android.text.Selection;
import android.view.View;
import android.view.ViewGroup;
import android.view.inputmethod.BaseInputConnection;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputConnection;
import android.view.inputmethod.InputMethodManager;

public final class TextInput extends View {
	// Calls into Go, registered by textinput.c: the field's whole state after the keyboard edited it.
	static native void edited(long field, int edit, String text, int selectionStart, int selectionEnd,
		int composingStart, int composingEnd);

	private static volatile TextInput current;

	private final Editable text = Editable.Factory.getInstance().newEditable("");
	private final InputMethodManager manager;
	private long field; // the field typed in, as Go names it; 0 while none is
	private int edit; // the keyboard's latest edit, counted
	private boolean secret;
	private int batch; // how deep the keyboard is in a batch of edits, told at its end
	private boolean fromGo; // the text is being set to Go's: nothing to tell Go

	private TextInput(Activity activity) {
		super(activity);
		manager = (InputMethodManager) activity.getSystemService(Context.INPUT_METHOD_SERVICE);
		setFocusable(true);
		setFocusableInTouchMode(true);
		// the screen reader reads the field Go draws, not this stand-in
		setImportantForAccessibility(IMPORTANT_FOR_ACCESSIBILITY_NO);
	}

	// install puts the view on the activity, a pixel in its corner, drawing nothing: Go draws the
	// field. Called by textinput.c whenever Android makes the activity.
	public static void install(final Activity activity) {
		activity.runOnUiThread(new Runnable() {
			@Override
			public void run() {
				TextInput input = new TextInput(activity);
				activity.addContentView(input, new ViewGroup.LayoutParams(1, 1));
				current = input;
			}
		});
	}

	// show brings the keyboard up for the view, or puts it away. Called from the drawing thread.
	public static void show(final boolean typing) {
		final TextInput input = current;
		if (input == null) return;
		input.post(new Runnable() {
			@Override
			public void run() {
				input.shown(typing);
			}
		});
	}

	private void shown(boolean typing) {
		if (manager == null) return;
		if (!typing) {
			field = 0;
			manager.hideSoftInputFromWindow(getWindowToken(), 0);
			clearFocus();
			return;
		}
		if (!hasFocus()) {
			requestFocus();
			manager.restartInput(this);
		}
		manager.showSoftInput(this, 0);
	}

	// set gives the view a field's text as Go has it, after the keyboard's edit numbered edit: the
	// field Go typed in last, or a field just focused (any edit). Indices are in UTF-16 units, as
	// Java counts; -1 composing means no composing word. Called from the drawing thread.
	public static void set(final long field, final int edit, final String text, final int selectionStart,
			final int selectionEnd, final int composingStart, final int composingEnd, final boolean secret) {
		final TextInput input = current;
		if (input == null) return;
		input.post(new Runnable() {
			@Override
			public void run() {
				input.took(field, edit, text, selectionStart, selectionEnd, composingStart, composingEnd, secret);
			}
		});
	}

	private void took(long field, int edit, String text, int selectionStart, int selectionEnd, int composingStart,
			int composingEnd, boolean secret) {
		boolean another = field != this.field || secret != this.secret;
		if (!another && edit != this.edit) return; // the keyboard typed since: Go's text is older
		boolean changed = another || !text.contentEquals(this.text);
		this.field = field;
		this.secret = secret;
		if (changed) this.text.replace(0, this.text.length(), text);
		Selection.setSelection(this.text, clamp(selectionStart), clamp(selectionEnd));
		BaseInputConnection.removeComposingSpans(this.text);
		if (composingStart >= 0 && composingEnd > composingStart) {
			fromGo = true;
			new Connection(this).setComposingRegion(clamp(composingStart), clamp(composingEnd));
			fromGo = false;
		}
		if (manager == null) return;
		if (changed) {
			manager.restartInput(this); // a text the keyboard did not type: it starts over with it
		} else {
			tellSelection();
		}
	}

	private int clamp(int index) {
		return Math.max(0, Math.min(index, text.length()));
	}

	// tellSelection tells the keyboard where the caret and the composing word are, as it expects
	// after every change: its suggestions follow them.
	private void tellSelection() {
		if (manager == null) return;
		manager.updateSelection(this, Selection.getSelectionStart(text), Selection.getSelectionEnd(text),
			BaseInputConnection.getComposingSpanStart(text), BaseInputConnection.getComposingSpanEnd(text));
	}

	// report tells Go the field's state after the keyboard edited it, and the keyboard the selection.
	private void report() {
		if (batch > 0 || field == 0 || fromGo) return;
		edit++;
		edited(field, edit, text.toString(), Selection.getSelectionStart(text), Selection.getSelectionEnd(text),
			BaseInputConnection.getComposingSpanStart(text), BaseInputConnection.getComposingSpanEnd(text));
		tellSelection();
	}

	@Override
	public boolean onCheckIsTextEditor() {
		return true;
	}

	@Override
	public InputConnection onCreateInputConnection(EditorInfo info) {
		info.inputType = InputType.TYPE_CLASS_TEXT | (secret ? InputType.TYPE_TEXT_VARIATION_PASSWORD
			: InputType.TYPE_TEXT_FLAG_AUTO_CORRECT);
		// one line: Enter is done; and no keyboard of its own over the whole screen when the phone lies
		info.imeOptions = EditorInfo.IME_ACTION_DONE | EditorInfo.IME_FLAG_NO_EXTRACT_UI | EditorInfo.IME_FLAG_NO_FULLSCREEN;
		info.initialSelStart = Selection.getSelectionStart(text);
		info.initialSelEnd = Selection.getSelectionEnd(text);
		return new Connection(this);
	}

	// Connection is how the keyboard edits the text: BaseInputConnection does the editing on the
	// view's copy; each edit is then told to Go.
	private final class Connection extends BaseInputConnection {
		Connection(View view) {
			super(view, true);
		}

		@Override
		public Editable getEditable() {
			return text;
		}

		@Override
		public boolean beginBatchEdit() {
			batch++;
			return true;
		}

		@Override
		public boolean endBatchEdit() {
			if (batch > 0) batch--;
			report();
			return batch > 0;
		}

		@Override
		public boolean commitText(CharSequence typed, int caret) {
			boolean done = super.commitText(typed, caret);
			report();
			return done;
		}

		@Override
		public boolean setComposingText(CharSequence typed, int caret) {
			boolean done = super.setComposingText(typed, caret);
			report();
			return done;
		}

		@Override
		public boolean setComposingRegion(int start, int end) {
			boolean done = super.setComposingRegion(start, end);
			report();
			return done;
		}

		@Override
		public boolean finishComposingText() {
			boolean done = super.finishComposingText();
			report();
			return done;
		}

		@Override
		public boolean deleteSurroundingText(int before, int after) {
			boolean done = super.deleteSurroundingText(before, after);
			report();
			return done;
		}

		@Override
		public boolean deleteSurroundingTextInCodePoints(int before, int after) {
			boolean done = super.deleteSurroundingTextInCodePoints(before, after);
			report();
			return done;
		}

		@Override
		public boolean setSelection(int start, int end) {
			boolean done = super.setSelection(start, end);
			report();
			return done;
		}
	}
}
