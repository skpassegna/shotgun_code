/**
 * Router Module for Shotgun Code Redesign
 *
 * A lightweight, custom screen-based navigation system for the Shotgun Code application.
 * This router manages navigation between 9 different screens in a linear workflow,
 * with support for breadcrumb navigation and back button functionality.
 *
 * Key Features:
 * - Screen-based navigation (no URL routing needed)
 * - Navigation history for back button
 * - Breadcrumb trail for quick navigation
 * - No external dependencies (no Vue Router)
 * - Simple and lightweight
 *
 * Screens:
 * 1. welcome - Landing page with onboarding
 * 2. folder - Folder selection
 * 3. files - File selection with search
 * 4. mode - Mode selection (Generate Code, Architecture Plan, etc.)
 * 5. task - Task description input
 * 6. review - Prompt review and token estimation
 * 7. execute - LLM execution (API call or manual)
 * 8. split - Diff splitting configuration
 * 9. apply - Apply patches
 *
 * @module router
 */

import { ref } from 'vue';

// ============================================================================
// State Management
// ============================================================================

/**
 * Current active screen
 * @type {Ref<string>}
 * @default 'welcome'
 */
export const currentScreen = ref('welcome');

/**
 * Navigation history stack for back button functionality
 * Stores previously visited screens in order
 * @type {Ref<Array<string>>}
 */
export const screenHistory = ref([]);

/**
 * Breadcrumb trail for navigation
 * Each breadcrumb contains a label and screen identifier
 * @type {Ref<Array<{label: string, screen: string}>>}
 */
export const breadcrumbs = ref([
  { label: 'Home', screen: 'welcome' }
]);

/**
 * Mapping of screen identifiers to human-readable labels
 * Used for breadcrumb display and UI labels
 * @type {Object<string, string>}
 */
const screenLabels = {
  'welcome': 'Home',
  'folder': 'Select Folder',
  'files': 'Choose Files',
  'mode': 'Choose Mode',
  'task': 'Describe Task',
  'review': 'Review Prompt',
  'execute': 'Execute',
  'split': 'Split Diff',
  'apply': 'Apply Patches'
};

// ============================================================================
// Navigation Functions
// ============================================================================

/**
 * Navigate to a new screen
 *
 * This is the primary navigation function used throughout the application.
 * It handles:
 * - Updating the current screen
 * - Adding to navigation history for back button
 * - Managing breadcrumb trail
 * - Preventing duplicate navigation
 *
 * @param {string} screen - Screen identifier (e.g., 'welcome', 'folder', 'files')
 * @param {boolean} addToBreadcrumb - Whether to add this screen to breadcrumb trail (default: true)
 *
 * @example
 * // Navigate to file selection screen
 * navigateTo('files');
 *
 * @example
 * // Navigate without adding to breadcrumbs (for temporary screens)
 * navigateTo('modal', false);
 */
export function navigateTo(screen, addToBreadcrumb = true) {
  // Validate input
  if (!screen || typeof screen !== 'string') {
    console.error('Invalid screen parameter:', screen);
    return;
  }

  // Validate screen exists in screenLabels
  if (!screenLabels[screen]) {
    console.error('Unknown screen:', screen);
    return;
  }

  // Prevent navigation to the same screen
  // This avoids duplicate entries in history
  if (currentScreen.value === screen) {
    return;
  }

  // Ensure screenHistory is an array
  if (!Array.isArray(screenHistory.value)) {
    screenHistory.value = [];
  }

  // Add current screen to history stack
  // This allows the back button to work correctly
  if (currentScreen.value) {
    screenHistory.value.push(currentScreen.value);
  }

  // Update the current screen
  currentScreen.value = screen;

  // Update breadcrumb trail if requested
  if (addToBreadcrumb) {
    // Ensure breadcrumbs is an array
    if (!Array.isArray(breadcrumbs.value)) {
      breadcrumbs.value = [{ label: 'Home', screen: 'welcome' }];
    }

    // Check if this screen is already in breadcrumbs
    // This happens when navigating back via breadcrumb click
    const existingIndex = breadcrumbs.value.findIndex(b => b && b.screen === screen);

    if (existingIndex !== -1) {
      // Screen exists in breadcrumbs - truncate to this point
      // This removes any screens that came after it
      breadcrumbs.value = breadcrumbs.value.slice(0, existingIndex + 1);
    } else {
      // Screen is new - add to breadcrumb trail
      const label = getScreenLabel(screen);
      if (label) {
        breadcrumbs.value.push({
          label,
          screen
        });
      }
    }
  }
}

/**
 * Navigate back to the previous screen
 *
 * Uses the navigation history stack to go back one screen.
 * Also updates the breadcrumb trail accordingly.
 * If there's no history (on welcome screen), this does nothing.
 *
 * @example
 * // Go back to previous screen
 * navigateBack();
 */
export function navigateBack() {
  // Ensure screenHistory is an array
  if (!Array.isArray(screenHistory.value)) {
    screenHistory.value = [];
    return;
  }

  // Check if there's any history to go back to
  if (screenHistory.value.length > 0) {
    // Pop the last screen from history
    const previousScreen = screenHistory.value.pop();

    // Validate previous screen
    if (previousScreen && typeof previousScreen === 'string' && screenLabels[previousScreen]) {
      // Update current screen to the previous one
      currentScreen.value = previousScreen;

      // Ensure breadcrumbs is an array
      if (!Array.isArray(breadcrumbs.value)) {
        breadcrumbs.value = [{ label: 'Home', screen: 'welcome' }];
      }

      // Update breadcrumbs by removing the last one
      // Keep at least one breadcrumb (Home)
      if (breadcrumbs.value.length > 1) {
        breadcrumbs.value.pop();
      }
    } else {
      // Invalid previous screen, reset to welcome
      console.error('Invalid previous screen:', previousScreen);
      resetNavigation();
    }
  }
}

/**
 * Navigate to a specific screen via breadcrumb click
 *
 * This function handles navigation when a user clicks on a breadcrumb.
 * It:
 * - Jumps to the selected screen
 * - Removes all screens after it from history
 * - Truncates breadcrumbs to the selected point
 *
 * @param {string} screen - Screen identifier from breadcrumb
 *
 * @example
 * // Jump back to folder selection screen
 * navigateToBreadcrumb('folder');
 */
export function navigateToBreadcrumb(screen) {
  // Validate input
  if (!screen || typeof screen !== 'string') {
    console.error('Invalid screen parameter:', screen);
    return;
  }

  // Validate screen exists
  if (!screenLabels[screen]) {
    console.error('Unknown screen:', screen);
    return;
  }

  // Ensure breadcrumbs is an array
  if (!Array.isArray(breadcrumbs.value)) {
    breadcrumbs.value = [{ label: 'Home', screen: 'welcome' }];
    return;
  }

  // Ensure screenHistory is an array
  if (!Array.isArray(screenHistory.value)) {
    screenHistory.value = [];
  }

  // Find the index of this screen in breadcrumbs
  const index = breadcrumbs.value.findIndex(b => b && b.screen === screen);

  if (index !== -1) {
    // Calculate how many screens to remove from history
    // This is the number of breadcrumbs after the selected one
    const screensToRemove = breadcrumbs.value.length - index - 1;

    // Remove those screens from history
    for (let i = 0; i < screensToRemove && screenHistory.value.length > 0; i++) {
      screenHistory.value.pop();
    }

    // Truncate breadcrumbs to the selected point
    breadcrumbs.value = breadcrumbs.value.slice(0, index + 1);

    // Update current screen
    currentScreen.value = screen;
  } else {
    // Screen not found in breadcrumbs, navigate normally
    console.warn('Screen not found in breadcrumbs:', screen);
    navigateTo(screen);
  }
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Get human-readable label for a screen identifier
 *
 * Converts screen identifiers (e.g., 'files') to display labels (e.g., 'Choose Files')
 * Used for breadcrumb display and UI labels.
 *
 * @param {string} screen - Screen identifier
 * @returns {string} Human-readable screen label
 *
 * @example
 * getScreenLabel('files'); // Returns: 'Choose Files'
 * getScreenLabel('unknown'); // Returns: 'unknown' (fallback)
 */
export function getScreenLabel(screen) {
  return screenLabels[screen] || screen;
}

/**
 * Reset navigation to initial state (welcome screen)
 *
 * This function:
 * - Returns to welcome screen
 * - Clears navigation history
 * - Resets breadcrumbs to just Home
 *
 * Useful for:
 * - Starting a new workflow
 * - Resetting after completion
 * - Error recovery
 *
 * @example
 * // Start fresh workflow
 * resetNavigation();
 */
export function resetNavigation() {
  try {
    currentScreen.value = 'welcome';
    screenHistory.value = [];
    breadcrumbs.value = [{ label: 'Home', screen: 'welcome' }];
  } catch (error) {
    console.error('Error resetting navigation:', error);
    // Force reset even if error occurs
    currentScreen.value = 'welcome';
    screenHistory.value = [];
    breadcrumbs.value = [{ label: 'Home', screen: 'welcome' }];
  }
}

/**
 * Check if back navigation is possible
 *
 * Returns true if there's at least one screen in the history stack.
 * Used to enable/disable the back button in the UI.
 *
 * @returns {boolean} True if can navigate back, false otherwise
 *
 * @example
 * // Conditionally show back button
 * if (canNavigateBack()) {
 *   showBackButton();
 * }
 */
export function canNavigateBack() {
  return screenHistory.value.length > 0;
}

/**
 * Get the complete navigation flow
 *
 * Returns an array of all screen identifiers in the standard workflow order.
 * Useful for:
 * - Determining next/previous screens
 * - Progress indicators
 * - Validation of workflow completion
 *
 * @returns {Array<string>} Array of screen identifiers in order
 */
export function getNavigationFlow() {
  return [
    'welcome',
    'folder',
    'files',
    'mode',
    'task',
    'review',
    'execute',
    'split',
    'apply'
  ];
}

/**
 * Get the next screen in the workflow
 *
 * Returns the next screen identifier based on the current screen.
 * Returns null if already at the last screen.
 *
 * @param {string} currentScreenId - Current screen identifier (optional, uses current if not provided)
 * @returns {string|null} Next screen identifier or null
 *
 * @example
 * // Get next screen after 'files'
 * const next = getNextScreen('files'); // Returns: 'mode'
 */
export function getNextScreen(currentScreenId = null) {
  const flow = getNavigationFlow();
  const screen = currentScreenId || currentScreen.value;
  const currentIndex = flow.indexOf(screen);

  if (currentIndex === -1 || currentIndex === flow.length - 1) {
    return null; // Not found or already at last screen
  }

  return flow[currentIndex + 1];
}

/**
 * Get the previous screen in the workflow
 *
 * Returns the previous screen identifier based on the current screen.
 * Returns null if already at the first screen.
 *
 * @param {string} currentScreenId - Current screen identifier (optional, uses current if not provided)
 * @returns {string|null} Previous screen identifier or null
 *
 * @example
 * // Get previous screen before 'files'
 * const prev = getPreviousScreen('files'); // Returns: 'folder'
 */
export function getPreviousScreen(currentScreenId = null) {
  const flow = getNavigationFlow();
  const screen = currentScreenId || currentScreen.value;
  const currentIndex = flow.indexOf(screen);

  if (currentIndex <= 0) {
    return null; // Not found or already at first screen
  }

  return flow[currentIndex - 1];
}

