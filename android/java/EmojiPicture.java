// What a go-renderer program asks Android to draw: an emoji the system's font files hold in a form
// Go does not read (COLRv1 outlines, a maker's own font). Android's Canvas draws whatever emoji font
// the phone has, so Go asks it once per emoji and size (emoji.c) and keeps the picture.
// go-ui-apk compiles it into the package.
package io.github.akrck02.gorenderer;

import android.graphics.Bitmap;
import android.graphics.Canvas;
import android.graphics.Color;
import android.graphics.Paint;
import android.graphics.Rect;
import java.util.Arrays;

public final class EmojiPicture {
	// The header before the pixels, as the Go side reads it (emoji.go): the picture's width and
	// height, then where it sits as float bits: left of the pen, top above the baseline, the advance.
	static final int WIDTH = 0, HEIGHT = 1, LEFT = 2, TOP = 3, ADVANCE = 4, HEADER = 5;

	// Pixels around the measured bounds, for the smoothed edges that fall outside them.
	static final int MARGIN = 1;

	// The selector asking for a character's emoji picture rather than its outline.
	static final String EMOJI_PRESENTATION = "\uFE0F";

	private EmojiPicture() {}

	// draw draws a character as a colour emoji at a size in pixels, and returns the header and its
	// ARGB pixels, not premultiplied; null when the phone has no colour emoji of it. It draws it in
	// two colours: a colour emoji keeps its own, and a letter in a plain font takes the paint's.
	public static int[] draw(int character, int pixels) {
		String text = new String(Character.toChars(character)) + EMOJI_PRESENTATION;
		Paint paint = new Paint(Paint.ANTI_ALIAS_FLAG);
		paint.setTextSize(pixels);
		float advance = paint.measureText(text);
		Rect bounds = new Rect();
		paint.getTextBounds(text, 0, text.length(), bounds);
		if (advance <= 0 || bounds.isEmpty()) return null;
		bounds.inset(-MARGIN, -MARGIN);
		int[] first = pixelsIn(text, paint, bounds, Color.RED);
		int[] second = pixelsIn(text, paint, bounds, Color.BLUE);
		if (!Arrays.equals(first, second) || isClear(first)) return null;
		int[] answer = new int[HEADER + first.length];
		answer[WIDTH] = bounds.width();
		answer[HEIGHT] = bounds.height();
		answer[LEFT] = Float.floatToIntBits(bounds.left);
		answer[TOP] = Float.floatToIntBits(-bounds.top);
		answer[ADVANCE] = Float.floatToIntBits(advance);
		System.arraycopy(first, 0, answer, HEADER, first.length);
		return answer;
	}

	// pixelsIn draws the text in a colour on a bitmap of the bounds, its pen where the bounds say.
	// getPixels gives them back not premultiplied, as Go keeps them.
	private static int[] pixelsIn(String text, Paint paint, Rect bounds, int colour) {
		Bitmap bitmap = Bitmap.createBitmap(bounds.width(), bounds.height(), Bitmap.Config.ARGB_8888);
		paint.setColor(colour);
		new Canvas(bitmap).drawText(text, -bounds.left, -bounds.top, paint);
		int[] pixels = new int[bounds.width() * bounds.height()];
		bitmap.getPixels(pixels, 0, bounds.width(), 0, 0, bounds.width(), bounds.height());
		bitmap.recycle();
		return pixels;
	}

	// isClear tells whether nothing was drawn.
	private static boolean isClear(int[] pixels) {
		for (int pixel : pixels) {
			if (Color.alpha(pixel) != 0) return false;
		}
		return true;
	}
}
