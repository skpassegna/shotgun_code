/**
 * useKeyboardShortcuts Composable
 * 
 * A Vue 3 composable for managing keyboard shortcuts in components.
 * Provides a clean, declarative way to register keyboard shortcuts
 * with automatic cleanup on component unmount.
 * 
 * Features:
 * - Support for Ctrl/Cmd, Shift, Alt modifiers
 * - Automatic event listener cleanup
 * - Prevents default browser behavior for registered shortcuts
 * - Cross-platform support (Ctrl on Windows/Linux, Cmd on macOS)
 * 
 * Usage:
 * ```javascript
 * import { useKeyboardShortcuts } from '@/composables/useKeyboardShortcuts';
 * 
 * useKeyboardShortcuts([
 *   { key: 'f', ctrl: true, handler: () => focusSearch() },
 *   { key: 'a', ctrl: true, handler: () => selectAll() },
 *   { key: 'i', ctrl: true, handler: () => invertSelection() },
 *   { key: 'Enter', ctrl: true, handler: () => nextScreen() },
 *   { key: 'Escape', handler: () => clearSearch() },
 * ]);
 * ```
 * 
 * @module useKeyboardShortcuts
 */

import { onMounted, onUnmounted } from 'vue';

/**
 * Register keyboard shortcuts for a component
 * 
 * @param {Array<Shortcut>} shortcuts - Array of shortcut configurations
 * @param {Object} options - Optional configuration
 * @param {boolean} options.enabled - Whether shortcuts are enabled (default: true)
 * @param {boolean} options.preventDefault - Whether to prevent default behavior (default: true)
 * 
 * @typedef {Object} Shortcut
 * @property {string} key - The key to listen for (e.g., 'f', 'Enter', 'Escape')
 * @property {boolean} [ctrl] - Whether Ctrl/Cmd must be pressed
 * @property {boolean} [shift] - Whether Shift must be pressed
 * @property {boolean} [alt] - Whether Alt must be pressed
 * @property {Function} handler - Function to call when shortcut is triggered
 * @property {string} [description] - Optional description for documentation
 */
export function useKeyboardShortcuts(shortcuts, options = {}) {
  const {
    enabled = true,
    preventDefault = true
  } = options;

  /**
   * Handle keydown events and match against registered shortcuts
   * 
   * @param {KeyboardEvent} event - The keyboard event
   */
  function handleKeyDown(event) {
    // Skip if shortcuts are disabled
    if (!enabled) return;

    // Skip if user is typing in an input field (unless explicitly allowed)
    const target = event.target;
    const isInputField = target.tagName === 'INPUT' || 
                        target.tagName === 'TEXTAREA' || 
                        target.isContentEditable;
    
    // Get the pressed key (normalized to lowercase for consistency)
    const key = event.key.toLowerCase();
    
    // Check modifier keys
    // Use metaKey (Cmd) on macOS, ctrlKey on Windows/Linux
    const ctrl = event.ctrlKey || event.metaKey;
    const shift = event.shiftKey;
    const alt = event.altKey;

    // Find matching shortcut
    for (const shortcut of shortcuts) {
      // Normalize shortcut key to lowercase
      const shortcutKey = shortcut.key.toLowerCase();
      
      // Check if key matches
      const keyMatches = key === shortcutKey;
      
      // Check if modifiers match
      const ctrlMatches = (shortcut.ctrl ?? false) === ctrl;
      const shiftMatches = (shortcut.shift ?? false) === shift;
      const altMatches = (shortcut.alt ?? false) === alt;
      
      // If all conditions match, trigger the handler
      if (keyMatches && ctrlMatches && shiftMatches && altMatches) {
        // Special handling for input fields
        // Only allow shortcuts with modifiers in input fields
        // This prevents interfering with normal typing
        if (isInputField && !ctrl && !alt && key.length === 1) {
          continue; // Skip this shortcut, allow normal typing
        }
        
        // Prevent default browser behavior if configured
        if (preventDefault) {
          event.preventDefault();
        }
        
        // Call the shortcut handler
        shortcut.handler(event);
        
        // Stop after first match
        break;
      }
    }
  }

  /**
   * Register event listener on component mount
   */
  onMounted(() => {
    window.addEventListener('keydown', handleKeyDown);
  });

  /**
   * Clean up event listener on component unmount
   */
  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeyDown);
  });

  /**
   * Return utility functions for programmatic control
   */
  return {
    /**
     * Get a formatted string representation of a shortcut
     * Useful for displaying shortcuts in UI
     * 
     * @param {Shortcut} shortcut - The shortcut to format
     * @returns {string} Formatted shortcut string (e.g., "Ctrl+F", "Ctrl+Shift+A")
     */
    formatShortcut(shortcut) {
      const parts = [];
      
      // Add modifiers
      if (shortcut.ctrl) {
        // Use Cmd symbol on macOS, Ctrl on other platforms
        const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
        parts.push(isMac ? '⌘' : 'Ctrl');
      }
      if (shortcut.shift) parts.push('Shift');
      if (shortcut.alt) {
        const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
        parts.push(isMac ? '⌥' : 'Alt');
      }
      
      // Add key (capitalize first letter)
      const key = shortcut.key.charAt(0).toUpperCase() + shortcut.key.slice(1);
      parts.push(key);
      
      return parts.join('+');
    },

    /**
     * Get all registered shortcuts with their formatted strings
     * Useful for displaying a shortcuts help dialog
     * 
     * @returns {Array<{shortcut: string, description: string}>}
     */
    getShortcutsList() {
      return shortcuts
        .filter(s => s.description) // Only include shortcuts with descriptions
        .map(s => ({
          shortcut: this.formatShortcut(s),
          description: s.description
        }));
    }
  };
}

/**
 * Common keyboard shortcuts for file selection screens
 * Pre-configured shortcuts that can be used across components
 * 
 * @param {Object} handlers - Object containing handler functions
 * @returns {Array<Shortcut>} Array of shortcut configurations
 */
export function useFileSelectionShortcuts(handlers) {
  return [
    {
      key: 'f',
      ctrl: true,
      handler: handlers.focusSearch,
      description: 'Focus search input'
    },
    {
      key: 'a',
      ctrl: true,
      handler: handlers.selectAll,
      description: 'Select all files'
    },
    {
      key: 'a',
      ctrl: true,
      shift: true,
      handler: handlers.deselectAll,
      description: 'Deselect all files'
    },
    {
      key: 'i',
      ctrl: true,
      handler: handlers.invertSelection,
      description: 'Invert selection'
    },
    {
      key: 'Enter',
      ctrl: true,
      handler: handlers.nextScreen,
      description: 'Go to next screen'
    },
    {
      key: 'Escape',
      handler: handlers.clearSearch,
      description: 'Clear search / Go back'
    }
  ];
}

/**
 * Common keyboard shortcuts for navigation
 * Pre-configured shortcuts for screen navigation
 * 
 * @param {Object} handlers - Object containing handler functions
 * @returns {Array<Shortcut>} Array of shortcut configurations
 */
export function useNavigationShortcuts(handlers) {
  return [
    {
      key: 'Enter',
      ctrl: true,
      handler: handlers.next,
      description: 'Next screen'
    },
    {
      key: 'Escape',
      handler: handlers.back,
      description: 'Previous screen'
    },
    {
      key: 'h',
      ctrl: true,
      handler: handlers.home,
      description: 'Go to home screen'
    }
  ];
}

/**
 * Keyboard shortcuts for text editing screens
 * Pre-configured shortcuts for prompt/text editing
 * 
 * @param {Object} handlers - Object containing handler functions
 * @returns {Array<Shortcut>} Array of shortcut configurations
 */
export function useTextEditingShortcuts(handlers) {
  return [
    {
      key: 's',
      ctrl: true,
      handler: handlers.save,
      description: 'Save changes'
    },
    {
      key: 'Enter',
      ctrl: true,
      handler: handlers.submit,
      description: 'Submit and continue'
    },
    {
      key: 'k',
      ctrl: true,
      handler: handlers.clearText,
      description: 'Clear text'
    }
  ];
}

/**
 * Keyboard shortcuts for copy/paste operations
 * Pre-configured shortcuts for clipboard operations
 * 
 * @param {Object} handlers - Object containing handler functions
 * @returns {Array<Shortcut>} Array of shortcut configurations
 */
export function useCopyPasteShortcuts(handlers) {
  return [
    {
      key: 'c',
      ctrl: true,
      shift: true,
      handler: handlers.copyAll,
      description: 'Copy all content'
    },
    {
      key: 'v',
      ctrl: true,
      shift: true,
      handler: handlers.pasteAndApply,
      description: 'Paste and apply'
    }
  ];
}

export default useKeyboardShortcuts;

