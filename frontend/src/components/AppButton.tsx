import React, { useRef } from 'react';
import { TouchableOpacity, Text, StyleSheet, ActivityIndicator, Animated } from 'react-native';
import { colors } from '../theme/colors';
import { typography } from '../theme/typography';

interface AppButtonProps {
  title: string;
  onPress: () => void;
  variant?: 'primary' | 'secondary' | 'success' | 'ghost';
  style?: any;
  textStyle?: any;
  loading?: boolean;
  disabled?: boolean;
}

export default function AppButton({ 
  title, 
  onPress, 
  variant = 'primary', 
  style,
  textStyle,
  loading = false,
  disabled = false
}: AppButtonProps) {
  const scale = useRef(new Animated.Value(1)).current;

  const handlePressIn = () => {
    Animated.spring(scale, {
      toValue: 0.96,
      useNativeDriver: true,
      speed: 50,
      bounciness: 4,
    }).start();
  };

  const handlePressOut = () => {
    Animated.spring(scale, {
      toValue: 1,
      useNativeDriver: true,
      speed: 50,
      bounciness: 4,
    }).start();
  };

  const getButtonStyle = () => {
    switch (variant) {
      case 'secondary': return styles.secondaryButton;
      case 'success':   return styles.successButton;
      case 'ghost':     return styles.ghostButton;
      default:          return styles.primaryButton;
    }
  };

  const getTextStyle = () => {
    switch (variant) {
      case 'secondary': return styles.secondaryText;
      case 'success':   return styles.successText;
      case 'ghost':     return styles.ghostText;
      default:          return styles.primaryText;
    }
  };

  return (
    <Animated.View style={[{ transform: [{ scale }] }, style]}>
      <TouchableOpacity 
        style={[
          styles.button, 
          getButtonStyle(),
          (disabled || loading) && styles.disabled,
        ]} 
        onPress={onPress}
        onPressIn={handlePressIn}
        onPressOut={handlePressOut}
        disabled={disabled || loading}
        activeOpacity={1}
      >
        {loading ? (
          <ActivityIndicator color={variant === 'primary' || variant === 'success' ? '#FFF' : colors.primary} size="small" />
        ) : (
          <Text style={[styles.text, getTextStyle(), textStyle]}>
            {title}
          </Text>
        )}
      </TouchableOpacity>
    </Animated.View>
  );
}

const styles = StyleSheet.create({
  button: {
    // minHeight + padding so the label still fits at large system font sizes.
    minHeight: 52,
    borderRadius: 20,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 24,
    paddingVertical: 12,
    width: '100%',
  },
  primaryButton: {
    backgroundColor: colors.primary,
  },
  secondaryButton: {
    backgroundColor: colors.surfaceWarm,
    borderWidth: 1,
    borderColor: colors.border,
  },
  successButton: {
    backgroundColor: colors.success,
  },
  ghostButton: {
    backgroundColor: 'transparent',
  },
  disabled: {
    opacity: 0.5,
  },
  text: {
    ...typography.body,
    fontWeight: '700',
    textAlign: 'center',
  },
  primaryText: {
    color: '#FFFFFF',
  },
  secondaryText: {
    color: colors.text,
  },
  successText: {
    color: '#FFFFFF',
  },
  ghostText: {
    color: colors.primary,
  },
});
