import React, { useState, useRef } from 'react';
import {
  TextInput, View, Text, StyleSheet, Animated,
  type BlurEvent, type FocusEvent,
} from 'react-native';
import { useScrollIntoView } from './KeyboardAwareScrollView';
import { colors } from '../theme/colors';
import { typography } from '../theme/typography';

interface AppTextInputProps extends React.ComponentProps<typeof TextInput> {
  label?: string;
  error?: string;
  icon?: string;
  /** Forwarded to the underlying TextInput, e.g. to focus the next field. */
  ref?: React.Ref<TextInput>;
}

export default function AppTextInput({ label, error, icon, ref, ...props }: AppTextInputProps) {
  const [isFocused, setIsFocused] = useState(false);
  const borderAnim = useRef(new Animated.Value(0)).current;
  const inputRef = useRef<TextInput>(null);
  const scrollIntoView = useScrollIntoView();

  // Keep our own handle for measuring while still honouring the caller's ref.
  const attachRef = (node: TextInput | null) => {
    inputRef.current = node;
    if (typeof ref === 'function') ref(node);
    else if (ref) (ref as React.RefObject<TextInput | null>).current = node;
  };

  const handleFocus = (event: FocusEvent) => {
    setIsFocused(true);
    Animated.timing(borderAnim, {
      toValue: 1,
      duration: 200,
      useNativeDriver: false,
    }).start();
    // Ask the surrounding scroll view to lift this field above the keyboard.
    scrollIntoView?.(inputRef.current);
    props.onFocus?.(event);
  };

  const handleBlur = (event: BlurEvent) => {
    setIsFocused(false);
    Animated.timing(borderAnim, {
      toValue: 0,
      duration: 200,
      useNativeDriver: false,
    }).start();
    props.onBlur?.(event);
  };

  const borderColor = borderAnim.interpolate({
    inputRange: [0, 1],
    outputRange: [error ? colors.error : colors.border, error ? colors.error : colors.primary],
  });

  return (
    <View style={styles.container}>
      {label && (
        <Text style={[styles.label, isFocused && styles.labelFocused]}>
          {label}
        </Text>
      )}
      <Animated.View style={[styles.inputContainer, { borderColor }]}>
        {icon && <Text style={styles.icon}>{icon}</Text>}
        <TextInput
          ref={attachRef}
          style={styles.input}
          placeholderTextColor={colors.textLight}
          {...props}
          onFocus={handleFocus}
          onBlur={handleBlur}
        />
      </Animated.View>
      {error && <Text style={styles.errorText}>⚠ {error}</Text>}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    marginBottom: 16,
    width: '100%',
  },
  label: {
    ...typography.caption,
    color: colors.textMuted,
    marginBottom: 8,
    marginLeft: 2,
  },
  labelFocused: {
    color: colors.primary,
  },
  inputContainer: {
    backgroundColor: colors.surfaceMid,
    borderWidth: 1.5,
    borderRadius: 14,
    // minHeight rather than height so the box grows instead of clipping the
    // text when the device is set to a large system font size.
    minHeight: 56,
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 4,
  },
  icon: {
    fontSize: 18,
    marginRight: 10,
  },
  input: {
    flex: 1,
    color: colors.text,
    ...typography.body,
    fontWeight: '500',
  },
  errorText: {
    ...typography.small,
    color: colors.error,
    marginTop: 6,
    marginLeft: 4,
  },
});
