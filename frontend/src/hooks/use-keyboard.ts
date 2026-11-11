import { useCallback, useEffect, useRef, useState } from 'react';
import { Keyboard, Platform, type KeyboardEvent, type LayoutChangeEvent } from 'react-native';

// iOS fires the "will" events early enough to animate alongside the keyboard.
// Android only has the "did" events.
const SHOW_EVENT = (Platform.OS === 'ios' ? 'keyboardWillShow' : 'keyboardDidShow') as 'keyboardDidShow';
const HIDE_EVENT = (Platform.OS === 'ios' ? 'keyboardWillHide' : 'keyboardDidHide') as 'keyboardDidHide';

/** Height of the on-screen keyboard, 0 when it is closed. */
export function useKeyboardHeight() {
  const [height, setHeight] = useState(0);

  useEffect(() => {
    const show = Keyboard.addListener(SHOW_EVENT, (e: KeyboardEvent) => {
      setHeight(e.endCoordinates.height);
    });
    const hide = Keyboard.addListener(HIDE_EVENT, () => setHeight(0));
    return () => {
      show.remove();
      hide.remove();
    };
  }, []);

  return height;
}

/** True while the keyboard is on screen — handy for collapsing decorative space. */
export function useKeyboardVisible() {
  return useKeyboardHeight() > 0;
}

/**
 * Measures how much of a container the keyboard genuinely covers.
 *
 * Android has been edge-to-edge since Expo SDK 54, so the window no longer
 * shrinks for the keyboard and `adjustResize` is a no-op — the keyboard just
 * sits on top of the UI. Instead of guessing per platform, this compares the
 * container's current height against its height with no keyboard: whatever the
 * OS already reclaimed is subtracted, and only the genuinely hidden pixels are
 * reported. That gives the same answer on iOS, on edge-to-edge Android, and on
 * any Android build that still resizes.
 *
 * Attach the returned `onLayout` to the container you want measured.
 */
export function useKeyboardOverlap(settleMs = 60) {
  const [overlap, setOverlap] = useState(0);
  const [keyboardHeight, setKeyboardHeight] = useState(0);

  const keyboardRef = useRef(0);
  const viewportRef = useRef(0);
  const restingRef = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Keyboard events and the resulting relayout race each other, so wait for
  // both to land before deciding how much is actually covered.
  const settle = useCallback(() => {
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => {
      const shrunkBy = Math.max(0, restingRef.current - viewportRef.current);
      const next = Math.max(0, Math.round(keyboardRef.current - shrunkBy));
      setOverlap((prev) => (Math.abs(prev - next) > 1 ? next : prev));
    }, settleMs);
  }, [settleMs]);

  useEffect(() => {
    const show = Keyboard.addListener(SHOW_EVENT, (e: KeyboardEvent) => {
      keyboardRef.current = e.endCoordinates.height;
      setKeyboardHeight(e.endCoordinates.height);
      settle();
    });
    const hide = Keyboard.addListener(HIDE_EVENT, () => {
      keyboardRef.current = 0;
      setKeyboardHeight(0);
      if (timer.current) clearTimeout(timer.current);
      setOverlap(0);
    });
    return () => {
      show.remove();
      hide.remove();
      if (timer.current) clearTimeout(timer.current);
    };
  }, [settle]);

  const onLayout = useCallback(
    (event: LayoutChangeEvent) => {
      viewportRef.current = event.nativeEvent.layout.height;
      if (keyboardRef.current === 0) {
        restingRef.current = viewportRef.current;
      } else {
        settle();
      }
    },
    [settle]
  );

  return { overlap, keyboardHeight, onLayout };
}
