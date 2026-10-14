// The one Java class a go-renderer program carries: what a screen reader (TalkBack) reads of the
// window. Android asks a view's AccessibilityNodeProvider, an abstract class, for what it holds,
// and JNI cannot make a subclass, so this is Java. Go draws everything in one native window; the
// program's interface tells this class what is where (update), and it answers Android with a
// virtual view for each thing, and passes back what the screen reader asks: press, scroll, the
// thing it is on. go-ui-apk compiles it into the package.
package io.github.akrck02.gorenderer;

import android.app.Activity;
import android.graphics.Rect;
import android.os.Build;
import android.os.Bundle;
import android.view.View;
import android.view.ViewGroup;
import android.view.ViewParent;
import android.view.accessibility.AccessibilityEvent;
import android.view.accessibility.AccessibilityManager;
import android.view.accessibility.AccessibilityNodeInfo;
import android.view.accessibility.AccessibilityNodeProvider;
import java.util.ArrayList;
import java.util.List;

// The calls Android 13 deprecated (obtain, setBoundsInParent) are the ones Android 8 has.
@SuppressWarnings("deprecation")
public final class ScreenReader extends AccessibilityNodeProvider {
	// What a node is and can do, as the Go side packs it (screenreader_nodes.go).
	static final int PRESSABLE = 1, LONG_PRESSABLE = 2, CHECKABLE = 4, CHECKED = 8, SELECTED = 16,
		DISABLED = 32, SCROLLABLE = 64, EDITABLE = 128, TYPING = 256, SECRET = 512, HEADING = 1024,
		RANGED = 2048;

	// What the screen reader asks, as the Go side numbers it.
	static final int PRESS = 1, LONG_PRESS = 2, FORWARD = 3, BACKWARD = 4;

	static final int NONE = Integer.MIN_VALUE; // no node: virtual view ids are never this

	// Calls into Go, registered by screenreader.c.
	static native void acted(int node, int action);
	static native void focused(int node);
	static native void readerChanged(boolean on);

	private static volatile ScreenReader current;

	private final View host;
	private volatile Nodes nodes = new Nodes(0);
	private int focusedNode = NONE; // the node the screen reader is on (its accessibility focus)
	private int hoveredNode = NONE; // the node under the finger exploring the screen

	private ScreenReader(View host) {
		this.host = host;
	}

	// Nodes is one frame's list: arrays side by side, read on the interface thread while the next
	// list is made on the drawing thread.
	static final class Nodes {
		final int[] ids, parents, flags;
		final float[] bounds, ranges; // left, top, right, bottom in window pixels; minimum, maximum, current
		final String[] classes, labels, values;

		Nodes(int count) {
			this(new int[count], new int[count], new int[count], new float[count * 4], new float[count * 3],
				new String[count], new String[count], new String[count]);
		}

		Nodes(int[] ids, int[] parents, int[] flags, float[] bounds, float[] ranges, String[] classes,
				String[] labels, String[] values) {
			this.ids = ids;
			this.parents = parents;
			this.flags = flags;
			this.bounds = bounds;
			this.ranges = ranges;
			this.classes = classes;
			this.labels = labels;
			this.values = values;
		}

		int indexOf(int id) {
			for (int index = 0; index < ids.length; index++) {
				if (ids[index] == id) return index;
			}
			return -1;
		}

		// at is the node under a point: the smallest that holds it, the later one of two alike, as
		// it is drawn over the other.
		int at(float x, float y) {
			int found = -1;
			float smallest = Float.MAX_VALUE;
			for (int index = 0; index < ids.length; index++) {
				float left = bounds[index * 4], top = bounds[index * 4 + 1];
				float right = bounds[index * 4 + 2], bottom = bounds[index * 4 + 3];
				if (x < left || x >= right || y < top || y >= bottom) continue;
				float area = (right - left) * (bottom - top);
				if (area <= smallest) {
					smallest = area;
					found = index;
				}
			}
			return found;
		}
	}

	// install puts a provider on the activity's content view, the one NativeActivity draws in, and
	// says whether a screen reader is on. Called by screenreader.c whenever Android makes the activity.
	public static void install(final Activity activity) {
		activity.runOnUiThread(new Runnable() {
			@Override
			public void run() {
				View host = contentView(activity);
				final ScreenReader reader = new ScreenReader(host);
				if (current != null) reader.nodes = current.nodes;
				current = reader;
				host.setImportantForAccessibility(View.IMPORTANT_FOR_ACCESSIBILITY_YES);
				host.setAccessibilityDelegate(new View.AccessibilityDelegate() {
					@Override
					public AccessibilityNodeProvider getAccessibilityNodeProvider(View view) {
						return reader;
					}
				});
				AccessibilityManager manager = (AccessibilityManager) activity.getSystemService(Activity.ACCESSIBILITY_SERVICE);
				if (manager == null) return;
				manager.addAccessibilityStateChangeListener(new AccessibilityManager.AccessibilityStateChangeListener() {
					@Override
					public void onAccessibilityStateChanged(boolean enabled) {
						readerChanged(enabled);
					}
				});
				readerChanged(manager.isEnabled());
			}
		});
	}

	private static View contentView(Activity activity) {
		View content = activity.findViewById(android.R.id.content);
		if (content instanceof ViewGroup && ((ViewGroup) content).getChildCount() > 0) {
			return ((ViewGroup) content).getChildAt(0);
		}
		return activity.getWindow().getDecorView();
	}

	// update takes the frame's nodes from the drawing thread. Go sends them only when they change.
	public static void update(int[] ids, int[] parents, int[] flags, float[] bounds, float[] ranges,
			String[] classes, String[] labels, String[] values) {
		final ScreenReader reader = current;
		if (reader == null) return;
		final Nodes updated = new Nodes(ids, parents, flags, bounds, ranges, classes, labels, values);
		reader.host.post(new Runnable() {
			@Override
			public void run() {
				reader.nodes = updated;
				if (reader.focusedNode != NONE && updated.indexOf(reader.focusedNode) < 0) {
					reader.focusedNode = NONE; // what it was on went away
					focused(0);
				}
				reader.send(View.NO_ID, AccessibilityEvent.TYPE_WINDOW_CONTENT_CHANGED);
			}
		});
	}

	// hover follows a finger exploring the screen, which Android sends as hover events while a
	// screen reader is on: the screen reader reads the node it comes onto. action is the
	// MotionEvent's; x and y are in window pixels.
	public static void hover(final int action, final float x, final float y) {
		final ScreenReader reader = current;
		if (reader == null) return;
		reader.host.post(new Runnable() {
			@Override
			public void run() {
				reader.hovered(action, x, y);
			}
		});
	}

	private void hovered(int action, float x, float y) {
		Nodes known = nodes;
		// the finger and the nodes are both in the window's pixels: the native surface is the window
		int index = action == android.view.MotionEvent.ACTION_HOVER_EXIT ? -1 : known.at(x, y);
		int node = index < 0 ? NONE : known.ids[index];
		if (node == hoveredNode) return;
		int previous = hoveredNode;
		hoveredNode = node;
		if (node != NONE) send(node, AccessibilityEvent.TYPE_VIEW_HOVER_ENTER);
		if (previous != NONE) send(previous, AccessibilityEvent.TYPE_VIEW_HOVER_EXIT);
	}

	@Override
	public AccessibilityNodeInfo createAccessibilityNodeInfo(int virtualViewId) {
		Nodes known = nodes;
		if (virtualViewId == View.NO_ID) return hostInfo(known);
		int index = known.indexOf(virtualViewId);
		if (index < 0) return null;
		AccessibilityNodeInfo info = AccessibilityNodeInfo.obtain(host, virtualViewId);
		info.setPackageName(host.getContext().getPackageName());
		info.setClassName(known.classes[index]);
		int parent = known.parents[index];
		if (parent == 0) {
			info.setParent(host);
		} else {
			info.setParent(host, parent);
		}
		for (int child = 0; child < known.ids.length; child++) {
			if (known.parents[child] == virtualViewId) info.addChild(host, known.ids[child]);
		}
		place(info, known, index);
		describe(info, known, index);
		return info;
	}

	private AccessibilityNodeInfo hostInfo(Nodes known) {
		AccessibilityNodeInfo info = AccessibilityNodeInfo.obtain(host);
		host.onInitializeAccessibilityNodeInfo(info);
		for (int index = 0; index < known.ids.length; index++) {
			if (known.parents[index] == 0) info.addChild(host, known.ids[index]);
		}
		return info;
	}

	// place says where a node is: on the screen, and inside the view. Its bounds are in the window's
	// pixels, as the program draws the whole window, while the view itself may start under the
	// status bar.
	private void place(AccessibilityNodeInfo info, Nodes known, int index) {
		Rect inWindow = new Rect((int) known.bounds[index * 4], (int) known.bounds[index * 4 + 1],
			(int) Math.ceil(known.bounds[index * 4 + 2]), (int) Math.ceil(known.bounds[index * 4 + 3]));
		int[] window = new int[2], view = new int[2];
		host.getRootView().getLocationOnScreen(window);
		host.getLocationInWindow(view);
		Rect inView = new Rect(inWindow);
		inView.offset(-view[0], -view[1]);
		info.setBoundsInParent(inView);
		Rect onScreen = new Rect(inWindow);
		onScreen.offset(window[0], window[1]);
		info.setBoundsInScreen(onScreen);
		info.setVisibleToUser(true);
	}

	// describe says what a node is called, its state and what can be done to it.
	private void describe(AccessibilityNodeInfo info, Nodes known, int index) {
		int flags = known.flags[index];
		String label = known.labels[index], value = known.values[index];
		boolean editable = (flags & EDITABLE) != 0;
		if (editable) {
			info.setEditable(true);
			info.setFocusable(true);
			info.setFocused((flags & TYPING) != 0);
			info.setPassword((flags & SECRET) != 0);
			info.setText(value);
			info.setHintText(label);
		} else {
			info.setText(label);
			if (!value.isEmpty()) {
				if (Build.VERSION.SDK_INT >= 30) {
					info.setStateDescription(value);
				} else {
					info.setText(label + ", " + value);
				}
			}
		}
		info.setEnabled((flags & DISABLED) == 0);
		info.setCheckable((flags & CHECKABLE) != 0);
		info.setChecked((flags & CHECKED) != 0);
		info.setSelected((flags & SELECTED) != 0);
		if (Build.VERSION.SDK_INT >= 28) info.setHeading((flags & HEADING) != 0);
		info.setImportantForAccessibility(true);
		if ((flags & RANGED) != 0) {
			info.setRangeInfo(AccessibilityNodeInfo.RangeInfo.obtain(AccessibilityNodeInfo.RangeInfo.RANGE_TYPE_FLOAT,
				known.ranges[index * 3], known.ranges[index * 3 + 1], known.ranges[index * 3 + 2]));
			info.addAction(AccessibilityNodeInfo.AccessibilityAction.ACTION_SET_PROGRESS);
		}
		if ((flags & PRESSABLE) != 0 && (flags & DISABLED) == 0) {
			info.setClickable(true);
			info.addAction(AccessibilityNodeInfo.AccessibilityAction.ACTION_CLICK);
		}
		if ((flags & LONG_PRESSABLE) != 0 && (flags & DISABLED) == 0) {
			info.setLongClickable(true);
			info.addAction(AccessibilityNodeInfo.AccessibilityAction.ACTION_LONG_CLICK);
		}
		if ((flags & SCROLLABLE) != 0) {
			info.setScrollable(true);
			info.addAction(AccessibilityNodeInfo.AccessibilityAction.ACTION_SCROLL_FORWARD);
			info.addAction(AccessibilityNodeInfo.AccessibilityAction.ACTION_SCROLL_BACKWARD);
		}
		boolean focused = known.ids[index] == focusedNode;
		info.setAccessibilityFocused(focused);
		info.addAction(focused ? AccessibilityNodeInfo.AccessibilityAction.ACTION_CLEAR_ACCESSIBILITY_FOCUS
			: AccessibilityNodeInfo.AccessibilityAction.ACTION_ACCESSIBILITY_FOCUS);
	}

	@Override
	public boolean performAction(int virtualViewId, int action, Bundle arguments) {
		if (virtualViewId == View.NO_ID) return host.performAccessibilityAction(action, arguments);
		Nodes known = nodes;
		int index = known.indexOf(virtualViewId);
		if (index < 0) return false;
		switch (action) {
		case AccessibilityNodeInfo.ACTION_ACCESSIBILITY_FOCUS:
			return focus(virtualViewId);
		case AccessibilityNodeInfo.ACTION_CLEAR_ACCESSIBILITY_FOCUS:
			if (focusedNode != virtualViewId) return false;
			focusedNode = NONE;
			focused(0);
			send(virtualViewId, AccessibilityEvent.TYPE_VIEW_ACCESSIBILITY_FOCUS_CLEARED);
			return true;
		case AccessibilityNodeInfo.ACTION_CLICK:
			acted(virtualViewId, PRESS);
			send(virtualViewId, AccessibilityEvent.TYPE_VIEW_CLICKED);
			return true;
		case AccessibilityNodeInfo.ACTION_LONG_CLICK:
			acted(virtualViewId, LONG_PRESS);
			send(virtualViewId, AccessibilityEvent.TYPE_VIEW_LONG_CLICKED);
			return true;
		case AccessibilityNodeInfo.ACTION_SCROLL_FORWARD:
			acted(virtualViewId, FORWARD);
			return true;
		case AccessibilityNodeInfo.ACTION_SCROLL_BACKWARD:
			acted(virtualViewId, BACKWARD);
			return true;
		}
		if (action == android.R.id.accessibilityActionSetProgress && arguments != null) {
			// the slider goes a step towards the value asked for
			float wanted = arguments.getFloat(AccessibilityNodeInfo.ACTION_ARGUMENT_PROGRESS_VALUE);
			float now = known.ranges[index * 3 + 2];
			if (wanted == now) return true;
			acted(virtualViewId, wanted > now ? FORWARD : BACKWARD);
			return true;
		}
		return false;
	}

	private boolean focus(int node) {
		if (focusedNode == node) return false;
		int previous = focusedNode;
		focusedNode = node;
		focused(node);
		if (previous != NONE) send(previous, AccessibilityEvent.TYPE_VIEW_ACCESSIBILITY_FOCUS_CLEARED);
		send(node, AccessibilityEvent.TYPE_VIEW_ACCESSIBILITY_FOCUSED);
		return true;
	}

	@Override
	public AccessibilityNodeInfo findFocus(int focus) {
		if (focus == AccessibilityNodeInfo.FOCUS_ACCESSIBILITY && focusedNode != NONE) {
			return createAccessibilityNodeInfo(focusedNode);
		}
		Nodes known = nodes;
		if (focus == AccessibilityNodeInfo.FOCUS_INPUT) {
			for (int index = 0; index < known.ids.length; index++) {
				if ((known.flags[index] & TYPING) != 0) return createAccessibilityNodeInfo(known.ids[index]);
			}
		}
		return null;
	}

	@Override
	public List<AccessibilityNodeInfo> findAccessibilityNodeInfosByText(String text, int virtualViewId) {
		List<AccessibilityNodeInfo> found = new ArrayList<AccessibilityNodeInfo>();
		Nodes known = nodes;
		String wanted = text.toLowerCase();
		for (int index = 0; index < known.ids.length; index++) {
			if (known.labels[index].toLowerCase().contains(wanted)) {
				found.add(createAccessibilityNodeInfo(known.ids[index]));
			}
		}
		return found;
	}

	// send tells the screen reader something happened to a node (or to the whole view, NO_ID).
	private void send(int node, int type) {
		AccessibilityManager manager = (AccessibilityManager) host.getContext().getSystemService(Activity.ACCESSIBILITY_SERVICE);
		if (manager == null || !manager.isEnabled()) return;
		ViewParent parent = host.getParent();
		if (parent == null) return;
		AccessibilityEvent event = AccessibilityEvent.obtain(type);
		event.setPackageName(host.getContext().getPackageName());
		if (node == View.NO_ID) {
			event.setSource(host);
			event.setContentChangeTypes(AccessibilityEvent.CONTENT_CHANGE_TYPE_SUBTREE);
		} else {
			event.setSource(host, node);
			Nodes known = nodes;
			int index = known.indexOf(node);
			if (index >= 0) {
				event.setClassName(known.classes[index]);
				event.getText().add(known.labels[index]);
			}
		}
		parent.requestSendAccessibilityEvent(host, event);
	}
}
