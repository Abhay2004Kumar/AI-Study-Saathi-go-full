import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
} from 'react';
import {
  Platform,
  ScrollView,
  StyleSheet,
  View,
  type LayoutChangeEvent,
  type NativeScrollEvent,
  type NativeSyntheticEvent,
  type ScrollViewProps,
} from 'react-native';
import { useKeyboardOverlap } from '../hooks/use-keyboard';

/** Anything with a native `measureInWindow` — a TextInput or View ref. */
export type Measurable = {
  measureInWindow: (
    callback: (x: number, y: number, width: number, height: number) => void
  ) => void;
};

type ScrollIntoView = (target: Measurable | null | undefined) => void;

const ScrollIntoViewContext = createContext<ScrollIntoView | null>(null);

/**
 * Lets a nested input ask the surrounding scroll view to reveal it above the
 * keyboard. Returns null when there is no KeyboardAwareScrollView above it.
 */
export function useScrollIntoView() {
  return useContext(ScrollIntoViewContext);
}

type Props = ScrollViewProps & {
  /** Gap kept between the focused input and the top of the keyboard. */
  extraScrollHeight?: number;
};

/**
 * A ScrollView that keeps the focused input visible above the keyboard.
 *
 * Pairs `useKeyboardOverlap` (which works out how many pixels the keyboard
 * really hides, edge-to-edge Android included) with a scroll that lifts the
 * focused field out from under it.
 */
export default function KeyboardAwareScrollView({
  extraScrollHeight = 24,
  contentContainerStyle,
  onLayout,
  onScroll,
  children,
  ...rest
}: Props) {
  const wrapperRef = useRef<View>(null);
  const scrollRef = useRef<ScrollView>(null);
  const scrollOffset = useRef(0);
  const focusedInput = useRef<Measurable | null>(null);

  const { overlap, keyboardHeight, onLayout: measureLayout } = useKeyboardOverlap();

  const revealFocusedInput = useCallback(() => {
    const target = focusedInput.current;
    const wrapper = wrapperRef.current;
    if (!target || !wrapper) return;

    wrapper.measureInWindow((_wx, wrapperY, _ww, wrapperHeight) => {
      target.measureInWindow((_x, inputY, _w, inputHeight) => {
        const visibleBottom = wrapperY + wrapperHeight - overlap;
        const distance = inputY + inputHeight + extraScrollHeight - visibleBottom;
        if (distance > 1) {
          scrollRef.current?.scrollTo({
            y: Math.max(0, scrollOffset.current + distance),
            animated: true,
          });
        }
      });
    });
  }, [extraScrollHeight, overlap]);

  // Re-run whenever the keyboard opens or the padding below the content
  // changes, since only then is there room to scroll the field into view.
  useEffect(() => {
    if (keyboardHeight === 0) {
      focusedInput.current = null;
      return;
    }
    revealFocusedInput();
  }, [keyboardHeight, overlap, revealFocusedInput]);

  const scrollIntoView = useCallback<ScrollIntoView>(
    (target) => {
      focusedInput.current = target ?? null;
      // Covers moving between fields while the keyboard is already open, where
      // neither the keyboard height nor the padding changes.
      setTimeout(revealFocusedInput, 80);
    },
    [revealFocusedInput]
  );

  const handleLayout = (event: LayoutChangeEvent) => {
    measureLayout(event);
    onLayout?.(event);
  };

  const handleScroll = (event: NativeSyntheticEvent<NativeScrollEvent>) => {
    scrollOffset.current = event.nativeEvent.contentOffset.y;
    onScroll?.(event);
  };

  const flattened = StyleSheet.flatten(contentContainerStyle) ?? {};
  const basePaddingBottom =
    typeof flattened.paddingBottom === 'number' ? flattened.paddingBottom : 0;

  return (
    <View ref={wrapperRef} style={styles.flex} onLayout={handleLayout} collapsable={false}>
      <ScrollIntoViewContext.Provider value={scrollIntoView}>
        <ScrollView
          ref={scrollRef}
          keyboardShouldPersistTaps="handled"
          keyboardDismissMode={Platform.OS === 'ios' ? 'interactive' : 'on-drag'}
          showsVerticalScrollIndicator={false}
          scrollEventThrottle={16}
          {...rest}
          onScroll={handleScroll}
          contentContainerStyle={[
            contentContainerStyle,
            { paddingBottom: basePaddingBottom + overlap },
          ]}
        >
          {children}
        </ScrollView>
      </ScrollIntoViewContext.Provider>
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
});
