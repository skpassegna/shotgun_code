<template>
  <div class="screen-container p-6 max-w-6xl mx-auto">
    <h1 class="text-3xl font-bold mb-6">Split Diff</h1>
    
    <div class="bg-white rounded-lg shadow p-6 mb-6">
      <p class="text-gray-600 mb-4">
        Configure how to split the generated diff into manageable chunks.
      </p>
      
      <div class="mb-6">
        <label class="block text-sm font-semibold mb-2">Lines per split:</label>
        <input
          v-model.number="linesPerSplit"
          type="number"
          min="100"
          max="2000"
          step="100"
          class="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
        <p class="text-xs text-gray-500 mt-1">
          Recommended: 500 lines. Adjust based on your LLM's context limit.
        </p>
      </div>

      <button
        @click="splitDiff"
        :disabled="isSplitting || !llmResponse"
        class="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <span v-if="isSplitting">⏳ Splitting...</span>
        <span v-else>Split Diff</span>
      </button>

      <!-- Split preview (shown after splitting) -->
      <div v-if="splitDiffs.length > 0" class="mt-6">
        <h3 class="font-semibold mb-3">Split Preview ({{ splitDiffs.length }} splits):</h3>
        <div class="space-y-3">
          <div
            v-for="(split, index) in splitDiffs"
            :key="index"
            class="p-4 border rounded-lg"
          >
            <div class="flex justify-between items-center mb-2">
              <span class="font-semibold">Split {{ index + 1 }} of {{ splitDiffs.length }}</span>
              <span class="text-sm text-gray-600">{{ countLines(split) }} lines</span>
            </div>
            <div class="flex gap-2">
              <button
                @click="togglePreview(index)"
                class="px-3 py-1 text-sm bg-gray-100 rounded hover:bg-gray-200"
              >
                {{ expandedSplits.has(index) ? 'Hide' : 'Preview' }}
              </button>
              <button
                @click="copySplit(split, index)"
                class="px-3 py-1 text-sm bg-blue-100 text-blue-700 rounded hover:bg-blue-200"
              >
                📋 Copy
              </button>
            </div>

            <!-- Expandable preview -->
            <div v-if="expandedSplits.has(index)" class="mt-3">
              <pre class="p-3 bg-gray-50 rounded text-xs overflow-x-auto max-h-96 overflow-y-auto">{{ split }}</pre>
            </div>
          </div>
        </div>
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
        class="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
      >
        Next →
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import { navigateTo, navigateBack } from '../router';
import { useAppStore } from '../stores/appStore';
import { useToast } from '../composables/useToast';

// Get Wails backend method
const SplitShotgunDiff = window.go?.main?.App?.SplitShotgunDiff;

// Get store and toast
const store = useAppStore();
const { showSuccess, showError } = useToast();

// Local state
const isSplitting = ref(false);
const expandedSplits = ref(new Set());

// Computed properties from store
const linesPerSplit = computed({
  get: () => store.linesPerSplit,
  set: (value) => store.setLinesPerSplit(value)
});

const llmResponse = computed(() => store.llmResponse);
const splitDiffs = computed(() => store.splitDiffs);

/**
 * Split the diff into manageable chunks
 */
async function splitDiff() {
  if (!SplitShotgunDiff) {
    showError('Backend not available. Please ensure the application is running properly.');
    return;
  }

  if (!llmResponse.value) {
    showError('No diff available to split. Please go back and execute the prompt.');
    return;
  }

  isSplitting.value = true;

  try {
    // Call backend to split diff
    const splits = await SplitShotgunDiff(llmResponse.value, linesPerSplit.value);

    // Save to store
    store.setSplitDiffs(splits);

    showSuccess(`Diff split into ${splits.length} chunks successfully!`);
  } catch (error) {
    console.error('Error splitting diff:', error);
    showError('Failed to split diff. Please try again.');
  } finally {
    isSplitting.value = false;
  }
}

/**
 * Count lines in a split
 */
function countLines(split) {
  return split.split('\n').length;
}

/**
 * Toggle preview for a split
 */
function togglePreview(index) {
  if (expandedSplits.value.has(index)) {
    expandedSplits.value.delete(index);
  } else {
    expandedSplits.value.add(index);
  }
  // Force reactivity
  expandedSplits.value = new Set(expandedSplits.value);
}

/**
 * Copy a split to clipboard
 */
async function copySplit(split, index) {
  try {
    await navigator.clipboard.writeText(split);
    showSuccess(`Split ${index + 1} copied to clipboard!`);
  } catch (error) {
    console.error('Failed to copy split:', error);
    showError('Failed to copy to clipboard.');
  }
}

function handleBack() {
  navigateBack();
}

function handleNext() {
  if (splitDiffs.value.length === 0) {
    showError('Please split the diff before proceeding.');
    return;
  }
  navigateTo('apply');
}
</script>

