import React from 'react';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { colors } from '../theme/colors';
import { typography } from '../theme/typography';

interface ActionButton {
  label: string;
  onPress: () => void;
  variant?: 'primary' | 'secondary' | 'danger';
  /** Sizes the button to its label instead of sharing the row equally. */
  compact?: boolean;
}

interface AppCardProps {
  title: string;
  subtitle?: string;
  onPress?: () => void;
  status?: 'UPLOADED' | 'PROCESSING' | 'READY' | 'FAILED';
  category?: 'SYLLABUS' | 'NOTES' | 'PYQ' | 'GENERAL';
  date?: string;
  actions?: ActionButton[];
}

const CATEGORY_EMOJI: Record<string, string> = {
  SYLLABUS: '📘',
  NOTES: '📝',
  PYQ: '🗞️',
  GENERAL: '📄',
};

export default function AppCard({ title, subtitle, onPress, status, category, date, actions }: AppCardProps) {
  const getStatusConfig = () => {
    switch (status) {
      case 'READY':      return { label: 'READY', bg: colors.successLight, color: colors.success, dot: '●' };
      case 'PROCESSING': return { label: 'PROCESSING', bg: '#FEF3C7', color: '#D97706', dot: '◌' };
      case 'FAILED':     return { label: 'FAILED', bg: '#FEE2E2', color: colors.error, dot: '●' };
      default:           return { label: 'UPLOADED', bg: colors.surfaceWarm, color: colors.textMuted, dot: '◌' };
    }
  };

  const statusConfig = status ? getStatusConfig() : null;

  return (
    <TouchableOpacity
      style={styles.card}
      onPress={onPress}
      disabled={!onPress}
      activeOpacity={0.85}
    >
      <View style={styles.topRow}>
        {/* Document icon */}
        <View style={styles.iconBox}>
          <Text style={styles.iconText}>{category ? CATEGORY_EMOJI[category] : '📄'}</Text>
        </View>

        <View style={styles.textGroup}>
          <Text style={styles.title} numberOfLines={1}>{title}</Text>
          {subtitle && (
            <Text style={styles.subtitle} numberOfLines={1}>{subtitle}</Text>
          )}
        </View>

        {statusConfig && (
          <View style={[styles.badge, { backgroundColor: statusConfig.bg }]}>
            <Text style={[styles.badgeDot, { color: statusConfig.color }]}>{statusConfig.dot} </Text>
            <Text style={[styles.badgeText, { color: statusConfig.color }]}>{statusConfig.label}</Text>
          </View>
        )}
      </View>

      {date && (
        <Text style={styles.date}>
          Added {new Date(date).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })}
        </Text>
      )}

      {actions && actions.length > 0 && (
        <View style={styles.actionsContainer}>
          {actions.map((action, index) => {
            const isDanger = action.variant === 'danger';
            const isPrimary = action.variant === 'primary' || (!action.variant && index === 0);
            return (
              <TouchableOpacity
                key={index}
                style={[
                  styles.actionBtn,
                  action.compact && styles.actionBtnCompact,
                  isPrimary && styles.actionBtnPrimary,
                  isDanger && styles.actionBtnDanger,
                  !isPrimary && !isDanger && styles.actionBtnSecondary,
                ]}
                onPress={action.onPress}
                activeOpacity={0.75}
              >
                <Text
                  style={[
                    styles.actionBtnText,
                    isPrimary && styles.actionBtnTextPrimary,
                    isDanger && styles.actionBtnTextDanger,
                  ]}
                  numberOfLines={1}
                  adjustsFontSizeToFit
                  minimumFontScale={0.8}
                >
                  {action.label}
                </Text>
              </TouchableOpacity>
            );
          })}
        </View>
      )}
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.surfaceWarm,
    borderRadius: 18,
    padding: 16,
    marginBottom: 14,
    borderWidth: 1,
    borderColor: colors.border,
  },
  topRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  iconBox: {
    width: 44,
    height: 44,
    borderRadius: 12,
    backgroundColor: colors.primaryLight,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  iconText: {
    fontSize: 22,
  },
  textGroup: {
    flex: 1,
  },
  title: {
    ...typography.subtitle,
    fontSize: 16,
    color: colors.text,
  },
  subtitle: {
    ...typography.small,
    color: colors.textMuted,
    marginTop: 2,
  },
  badge: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 20,
    marginLeft: 8,
  },
  badgeDot: {
    fontSize: 8,
  },
  badgeText: {
    ...typography.small,
    fontSize: 10,
    fontWeight: '700',
  },
  date: {
    ...typography.small,
    color: colors.textLight,
    marginTop: 10,
    marginLeft: 56,
  },
  actionsContainer: {
    flexDirection: 'row',
    marginTop: 14,
    paddingTop: 14,
    borderTopWidth: 1,
    borderTopColor: colors.divider,
    gap: 6,
  },
  actionBtn: {
    flex: 1,
    minWidth: 0,
    paddingVertical: 9,
    paddingHorizontal: 6,
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
  actionBtnCompact: {
    flex: 0,
    paddingHorizontal: 14,
  },
  actionBtnPrimary: {
    backgroundColor: colors.primary,
  },
  actionBtnSecondary: {
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border,
  },
  actionBtnDanger: {
    backgroundColor: '#FEE2E2',
  },
  actionBtnText: {
    ...typography.caption,
    fontWeight: '700',
    color: colors.textMuted,
  },
  actionBtnTextPrimary: {
    color: '#FFFFFF',
  },
  actionBtnTextDanger: {
    color: colors.error,
  },
});
