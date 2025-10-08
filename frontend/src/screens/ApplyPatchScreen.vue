<template>
  <div class="screen-container p-6 max-w-6xl mx-auto">
    <h1 class="text-3xl font-bold mb-6">Apply Patches</h1>
    
    <div class="bg-white rounded-lg shadow p-6 mb-6">
      <p class="text-gray-600 mb-6">
        Apply the split diffs to your codebase. Copy each split and apply manually, or use automatic patching.
      </p>
      
      <!-- Split list -->
      <div v-if="splitDiffs.length > 0" class="space-y-4">
        <div
          v-for="(split, index) in splitDiffs"
          :key="index"
          class="border rounded-lg p-4"
          :class="{ 'border-green-500 bg-green-50': appliedSplits.has(index) }"
        >
          <div class="flex justify-between items-center mb-3">
            <div>
              <h3 class="font-semibold flex items-center gap-2">
                Split {{ index + 1 }} of {{ splitDiffs.length }}
                <span v-if="appliedSplits.has(index)" class="text-green-600 text-sm">✓ Applied</span>
              </h3>
              <div class="flex gap-4 text-sm text-gray-600 mt-1">
                <span>{{ countLines(split) }} lines</span>
              </div>
            </div>
            <div class="flex gap-2">
              <button
                @click="copySplit(split, index)"
                class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
              >
                📋 Copy
              </button>
              <button
                v-if="!appliedSplits.has(index)"
                @click="markAsApplied(index)"
                class="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 transition-colors"
              >
                ✓ Mark Applied
              </button>
            </div>
          </div>

          <details class="mt-3">
            <summary class="cursor-pointer text-sm text-blue-600 hover:underline">
              Show diff preview
            </summary>
            <pre class="mt-2 p-3 bg-gray-50 rounded text-xs overflow-x-auto max-h-96 overflow-y-auto">{{ split }}</pre>
          </details>
        </div>
      </div>

      <div v-else class="p-4 bg-gray-50 rounded border border-dashed">
        <p class="text-sm text-gray-600 text-center">
          No splits available. Please go back and split the diff.
        </p>
      </div>
      
      <!-- Coming soon features -->
      <div class="mt-6 p-4 bg-yellow-50 rounded border border-yellow-200">
        <p class="text-sm font-semibold mb-2">🚧 Coming Soon:</p>
        <ul class="text-sm text-gray-700 space-y-1">
          <li>• Automatic patch application</li>
          <li>• Conflict detection and resolution</li>
          <li>• Rollback functionality</li>
        </ul>
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
        @click="handleFinish" 
        class="px-6 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
      >
        ✓ Finish
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import { navigateBack, resetNavigation } from '../router';
import { useAppStore } from '../stores/appStore';
import { useToast } from '../composables/useToast';

// Get store and toast
const store = useAppStore();
const { showSuccess, showInfo } = useToast();

// Local state
const appliedSplits = ref(new Set());

// Computed properties from store
const splitDiffs = computed(() => store.splitDiffs);

/**
 * Count lines in a split
 */
function countLines(split) {
  return split.split('\n').length;
}

/**
 * Copy a split to clipboard
 */
async function copySplit(split, index) {
  try {
    await navigator.clipboard.writeText(split);
    showSuccess(`Split ${index + 1} copied to clipboard! Apply it to your codebase.`);
  } catch (error) {
    console.error('Failed to copy split:', error);
    showError('Failed to copy to clipboard.');
  }
}

/**
 * Mark a split as applied
 */
function markAsApplied(index) {
  appliedSplits.value.add(index);
  // Force reactivity
  appliedSplits.value = new Set(appliedSplits.value);
  showSuccess(`Split ${index + 1} marked as applied!`);
}

function handleBack() {
  navigateBack();
}

function handleFinish() {
  const totalSplits = splitDiffs.value.length;
  const appliedCount = appliedSplits.value.size;

  if (appliedCount < totalSplits) {
    showInfo(`You've applied ${appliedCount} of ${totalSplits} splits. You can continue later.`);
  } else {
    showSuccess('All splits applied! Great work!');
  }

  // Reset store and navigation
  store.resetStore();
  resetNavigation();
}
</script>

