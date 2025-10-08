<template>
  <div class="screen-container p-6 max-w-6xl mx-auto">
    <h1 class="text-3xl font-bold mb-6">Review Prompt</h1>

    <div v-if="isGenerating" class="bg-white rounded-lg shadow p-6 mb-6 text-center">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
      <p class="text-gray-600">Generating context and composing prompt...</p>
      <p class="text-sm text-gray-500 mt-2">This may take a few moments for large codebases...</p>

      <!-- Show cancel button after 5 minutes -->
      <div v-if="showCancelButton" class="mt-4">
        <p class="text-sm text-orange-600 mb-2">⏱️ This is taking longer than expected...</p>
        <button
          @click="cancelGeneration"
          class="px-6 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
        >
          ✕ Cancel Operation
        </button>
      </div>
    </div>

    <div v-else-if="hasError" class="bg-white rounded-lg shadow p-6 mb-6">
      <div class="text-center">
        <div class="text-red-600 text-5xl mb-4">⚠️</div>
        <h2 class="text-xl font-semibold text-red-600 mb-2">Generation Failed</h2>
        <p class="text-gray-600 mb-4">{{ errorMessage }}</p>
        <button
          @click="retryGeneration"
          class="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          🔄 Retry
        </button>
      </div>
    </div>

    <div v-else class="bg-white rounded-lg shadow p-6 mb-6">
      <p class="text-gray-600 mb-4">
        Review your final prompt before execution. Check token count and estimated cost.
      </p>

      <!-- Token estimation -->
      <div class="mb-4 p-4 bg-gray-50 rounded border">
        <div class="flex justify-between items-center mb-2">
          <span class="font-semibold">Token Estimation:</span>
          <span class="text-lg font-bold text-blue-600">~{{ safeEstimatedTokens.toLocaleString() }} tokens</span>
        </div>
        <div class="flex justify-between items-center text-sm text-gray-600">
          <span>Estimated cost ({{ safeLLMProvider }}):</span>
          <span v-if="safeEstimatedCost > 0">~${{ safeEstimatedCost.toFixed(4) }}</span>
          <span v-else>Unknown (custom provider)</span>
        </div>
      </div>

      <!-- Prompt preview -->
      <div class="mb-4">
        <label class="block text-sm font-semibold mb-2">Final Prompt:</label>
        <textarea
          v-model="composedPrompt"
          readonly
          class="w-full h-64 px-4 py-3 border rounded-lg bg-gray-50 font-mono text-sm resize-none"
          placeholder="Your composed prompt will appear here..."
        ></textarea>
      </div>

      <!-- Copy button -->
      <div class="flex gap-2">
        <button
          @click="copyEntirePrompt"
          class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="!hasValidPrompt"
        >
          📋 Copy Entire Prompt
        </button>
        <button
          @click="regeneratePrompt"
          class="px-4 py-2 bg-gray-100 text-gray-700 rounded hover:bg-gray-200 transition-colors"
        >
          🔄 Regenerate
        </button>
      </div>
    </div>

    <div class="flex justify-between mt-8">
      <button
        @click="handleBack"
        class="px-6 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors"
      >
        ← Back
      </button>
      <button
        @click="handleNext"
        class="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="!canProceed"
      >
        Execute →
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { navigateTo, navigateBack } from '../router';
import { useAppStore } from '../stores/appStore';
import { useToast } from '../composables/useToast';

// Get Wails backend methods
const RequestShotgunContextGeneration = window.go?.main?.App?.RequestShotgunContextGeneration;
const CancelShotgunContextGeneration = window.go?.main?.App?.CancelShotgunContextGeneration;
const GeneratePrompt = window.go?.main?.App?.GeneratePrompt;
const EstimateTokens = window.go?.main?.App?.EstimateTokens;
const EstimateCost = window.go?.main?.App?.EstimateCost;

// Get store and toast
const store = useAppStore();
const { showSuccess, showError, showInfo } = useToast();

// Local state
const isGenerating = ref(false);
const hasError = ref(false);
const errorMessage = ref('');
const showCancelButton = ref(false);
const cancelButtonTimeout = ref(null);
const eventCleanup = ref(null);

// Computed properties from store with null safety
const composedPrompt = computed(() => {
  const prompt = store.composedPrompt;
  return (prompt !== null && prompt !== undefined && typeof prompt === 'string') ? prompt : '';
});

const estimatedTokens = computed(() => {
  const tokens = store.estimatedTokens;
  return tokens;
});

const estimatedCost = computed(() => {
  const cost = store.estimatedCost;
  return cost;
});

const llmProvider = computed(() => {
  const provider = store.llmProvider;
  return (provider !== null && provider !== undefined && typeof provider === 'string') ? provider : 'unknown';
});

// Safe computed properties with type validation
const safeEstimatedTokens = computed(() => {
  const tokens = estimatedTokens.value;
  const numTokens = Number(tokens);
  return (!isNaN(numTokens) && numTokens >= 0) ? Math.floor(numTokens) : 0;
});

const safeEstimatedCost = computed(() => {
  const cost = estimatedCost.value;
  const numCost = Number(cost);
  return (!isNaN(numCost) && numCost >= 0) ? numCost : 0;
});

const safeLLMProvider = computed(() => {
  return llmProvider.value || 'unknown';
});

// Validation computed properties
const hasValidPrompt = computed(() => {
  const prompt = composedPrompt.value;
  return prompt !== null && prompt !== undefined && typeof prompt === 'string' && prompt.trim().length > 0;
});

const canProceed = computed(() => {
  return hasValidPrompt.value && !isGenerating.value && !hasError.value;
});

/**
 * Cleanup event listeners and timeouts
 */
function cleanupListeners() {
  // Clear cancel button timeout if exists
  if (cancelButtonTimeout.value) {
    clearTimeout(cancelButtonTimeout.value);
    cancelButtonTimeout.value = null;
  }

  // Hide cancel button
  showCancelButton.value = false;

  // Cleanup event listener if exists
  if (eventCleanup.value && typeof eventCleanup.value === 'function') {
    eventCleanup.value();
    eventCleanup.value = null;
  }

  // Also try to remove event listeners directly
  if (window.runtime?.EventsOff) {
    window.runtime.EventsOff('shotgunContextGenerated');
    window.runtime.EventsOff('shotgunContextError');
  }
}

/**
 * Generate context and compose prompt
 */
async function generateContextAndPrompt() {
  // Validate backend availability
  if (!RequestShotgunContextGeneration || !GeneratePrompt || !EstimateTokens || !EstimateCost) {
    hasError.value = true;
    errorMessage.value = 'Backend not available. Please ensure the application is running properly.';
    showError(errorMessage.value);
    return;
  }

  // Validate required data
  if (!store.projectFolder || typeof store.projectFolder !== 'string' || store.projectFolder.trim() === '') {
    hasError.value = true;
    errorMessage.value = 'No project folder selected. Please go back and select a folder.';
    showError(errorMessage.value);
    return;
  }

  if (!store.selectedMode || typeof store.selectedMode !== 'string' || store.selectedMode.trim() === '') {
    hasError.value = true;
    errorMessage.value = 'No mode selected. Please go back and select a mode.';
    showError(errorMessage.value);
    return;
  }

  if (!store.taskDescription || typeof store.taskDescription !== 'string' || store.taskDescription.trim() === '') {
    hasError.value = true;
    errorMessage.value = 'No task description provided. Please go back and describe your task.';
    showError(errorMessage.value);
    return;
  }

  // Reset state
  isGenerating.value = true;
  hasError.value = false;
  errorMessage.value = '';

  // Cleanup any existing listeners
  cleanupListeners();

  try {
    // Get excluded paths with null safety
    const excludedPaths = Array.isArray(store.excludedFilePaths) ? store.excludedFilePaths : [];

    // Show cancel button after 5 minutes (no timeout, just UI feedback)
    cancelButtonTimeout.value = setTimeout(() => {
      if (isGenerating.value) {
        showCancelButton.value = true;
        showInfo('Generation is taking longer than expected. You can cancel if needed.');
      }
    }, 5 * 60 * 1000); // 5 minutes

    // Listen for context generation completion (FIXED: correct event name)
    eventCleanup.value = window.runtime?.EventsOn('shotgunContextGenerated', async (context) => {
      try {
        // Clear cancel button timeout
        if (cancelButtonTimeout.value) {
          clearTimeout(cancelButtonTimeout.value);
          cancelButtonTimeout.value = null;
        }
        showCancelButton.value = false;

        // Validate context
        if (!context || typeof context !== 'string' || context.trim() === '') {
          throw new Error('Generated context is empty or invalid');
        }

        // Save context to store
        store.setGeneratedContext(context);

        // Step 2: Generate prompt from context
        const prompt = await GeneratePrompt(
          context,
          store.selectedMode,
          store.taskDescription,
          store.customRules || ''
        );

        // Validate prompt
        if (!prompt || typeof prompt !== 'string' || prompt.trim() === '') {
          throw new Error('Generated prompt is empty or invalid');
        }

        // Save prompt to store
        store.setComposedPrompt(prompt);

        // Step 3: Estimate tokens
        const tokens = await EstimateTokens(prompt);

        // Validate tokens
        const numTokens = Number(tokens);
        if (isNaN(numTokens) || numTokens < 0) {
          console.warn('Invalid token count received:', tokens);
          store.setEstimatedTokens(0);
        } else {
          store.setEstimatedTokens(numTokens);
        }

        // Step 4: Estimate cost (assume 1/4 of input tokens for output)
        const outputTokens = Math.floor(Math.max(0, numTokens) / 4);
        const modelName = (store.llmModel && typeof store.llmModel === 'string' && store.llmModel.trim() !== '')
          ? store.llmModel
          : 'gemini-2.5-flash';

        const cost = await EstimateCost(
          store.llmProvider || 'google',
          modelName,
          Math.max(0, numTokens),
          outputTokens
        );

        // Validate cost
        const numCost = Number(cost);
        if (isNaN(numCost) || numCost < 0) {
          console.warn('Invalid cost received:', cost);
          store.setEstimatedCost(0);
        } else {
          store.setEstimatedCost(numCost);
        }

        isGenerating.value = false;
        hasError.value = false;
        showSuccess('Prompt generated successfully!');

        // Cleanup event listener
        cleanupListeners();
      } catch (error) {
        console.error('Error processing context:', error);
        cleanupListeners();
        isGenerating.value = false;
        hasError.value = true;
        errorMessage.value = error.message || 'Failed to generate prompt. Please try again.';
        showError(errorMessage.value);
      }
    });

    // Listen for context generation errors
    const errorCleanup = window.runtime?.EventsOn('shotgunContextError', (error) => {
      console.error('Context generation error:', error);
      cleanupListeners();
      isGenerating.value = false;
      hasError.value = true;
      errorMessage.value = typeof error === 'string' ? error : 'Failed to generate context. Please try again.';
      showError(errorMessage.value);
    });

    // Request context generation (this runs as a background job)
    RequestShotgunContextGeneration(store.projectFolder, excludedPaths);

  } catch (error) {
    console.error('Error generating context:', error);
    cleanupListeners();
    isGenerating.value = false;
    hasError.value = true;
    errorMessage.value = error.message || 'Failed to generate context. Please try again.';
    showError(errorMessage.value);
  }
}

/**
 * Cancel the ongoing context generation
 */
async function cancelGeneration() {
  try {
    // Call backend to cancel the generation
    if (CancelShotgunContextGeneration && typeof CancelShotgunContextGeneration === 'function') {
      await CancelShotgunContextGeneration();
      showInfo('Context generation cancelled successfully.');
    } else {
      console.warn('CancelShotgunContextGeneration method not available');
      showInfo('Generation cancelled (backend method not available).');
    }
  } catch (error) {
    console.error('Error cancelling generation:', error);
    // Continue with cleanup even if backend call fails
  } finally {
    // Clean up listeners and state
    cleanupListeners();
    isGenerating.value = false;
    hasError.value = true;
    errorMessage.value = 'Context generation was cancelled by user.';
  }
}

/**
 * Copy entire prompt to clipboard with validation
 */
async function copyEntirePrompt() {
  // Validate prompt exists and is not empty
  if (!hasValidPrompt.value) {
    showError('No prompt available to copy. Please generate the prompt first.');
    return;
  }

  const promptText = composedPrompt.value;

  // Additional validation
  if (typeof promptText !== 'string' || promptText.trim().length === 0) {
    showError('Prompt is empty or invalid. Cannot copy.');
    return;
  }

  try {
    // Try using the Clipboard API
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(promptText);
      showSuccess(`Prompt copied to clipboard! (${safeEstimatedTokens.value.toLocaleString()} tokens)`);
    } else {
      // Fallback for older browsers
      const textArea = document.createElement('textarea');
      textArea.value = promptText;
      textArea.style.position = 'fixed';
      textArea.style.left = '-999999px';
      textArea.style.top = '-999999px';
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();

      try {
        const successful = document.execCommand('copy');
        if (successful) {
          showSuccess(`Prompt copied to clipboard! (${safeEstimatedTokens.value.toLocaleString()} tokens)`);
        } else {
          throw new Error('execCommand failed');
        }
      } finally {
        document.body.removeChild(textArea);
      }
    }
  } catch (error) {
    console.error('Failed to copy prompt:', error);
    showError('Failed to copy to clipboard. Please try selecting and copying manually.');
  }
}

/**
 * Regenerate the prompt
 */
function regeneratePrompt() {
  if (isGenerating.value) {
    showInfo('Generation already in progress...');
    return;
  }

  showInfo('Regenerating prompt...');
  generateContextAndPrompt();
}

/**
 * Retry generation after error
 */
function retryGeneration() {
  hasError.value = false;
  errorMessage.value = '';
  generateContextAndPrompt();
}

function handleBack() {
  // Cleanup listeners before navigating away
  cleanupListeners();
  navigateBack();
}

function handleNext() {
  // Validate before proceeding
  if (!hasValidPrompt.value) {
    showError('Please generate a valid prompt before proceeding.');
    return;
  }

  if (isGenerating.value) {
    showError('Please wait for prompt generation to complete.');
    return;
  }

  navigateTo('execute');
}

// Generate context and prompt when component mounts
onMounted(() => {
  // Only generate if we don't have a valid prompt
  if (!hasValidPrompt.value) {
    generateContextAndPrompt();
  }
});

// Cleanup on unmount
onBeforeUnmount(() => {
  cleanupListeners();
});
</script>

