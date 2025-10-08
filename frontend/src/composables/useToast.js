/**
 * Toast Notification Composable
 * 
 * Provides a simple toast notification system for displaying success, error,
 * warning, and info messages to the user.
 * 
 * Features:
 * - Multiple toast types (success, error, warning, info)
 * - Auto-dismiss after configurable duration
 * - Manual dismiss
 * - Queue management (multiple toasts)
 * - Reactive state
 * 
 * Usage:
 * ```js
 * import { useToast } from '@/composables/useToast';
 * 
 * const { showSuccess, showError, showWarning, showInfo } = useToast();
 * 
 * showSuccess('File saved successfully!');
 * showError('Failed to load file tree');
 * ```
 */

import { ref } from 'vue';

// Global toast state (shared across all components)
const toasts = ref([]);
let nextId = 0;

/**
 * Toast notification composable
 */
export function useToast() {
  /**
   * Add a toast notification
   * 
   * @param {string} message - The message to display
   * @param {string} type - Toast type: 'success', 'error', 'warning', 'info'
   * @param {number} duration - Duration in milliseconds (0 = no auto-dismiss)
   */
  function addToast(message, type = 'info', duration = 3000) {
    const id = nextId++;
    const toast = {
      id,
      message,
      type,
      visible: true,
    };
    
    toasts.value.push(toast);
    
    // Auto-dismiss after duration
    if (duration > 0) {
      setTimeout(() => {
        removeToast(id);
      }, duration);
    }
    
    return id;
  }
  
  /**
   * Remove a toast by ID
   */
  function removeToast(id) {
    const index = toasts.value.findIndex(t => t.id === id);
    if (index !== -1) {
      toasts.value.splice(index, 1);
    }
  }
  
  /**
   * Clear all toasts
   */
  function clearAllToasts() {
    toasts.value = [];
  }
  
  /**
   * Show success toast
   */
  function showSuccess(message, duration = 3000) {
    return addToast(message, 'success', duration);
  }
  
  /**
   * Show error toast
   */
  function showError(message, duration = 5000) {
    return addToast(message, 'error', duration);
  }
  
  /**
   * Show warning toast
   */
  function showWarning(message, duration = 4000) {
    return addToast(message, 'warning', duration);
  }
  
  /**
   * Show info toast
   */
  function showInfo(message, duration = 3000) {
    return addToast(message, 'info', duration);
  }
  
  return {
    toasts,
    addToast,
    removeToast,
    clearAllToasts,
    showSuccess,
    showError,
    showWarning,
    showInfo,
  };
}

