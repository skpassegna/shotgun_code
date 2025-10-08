<!--
  JobQueueStatus Component
  
  Displays a floating panel showing active background jobs with real-time progress updates.
  This component listens to Wails events for job queue updates and displays running jobs
  with progress bars and cancel buttons.
  
  Features:
  - Real-time job status updates via Wails events
  - Progress bars for running jobs
  - Cancel button for active jobs
  - Auto-hide when no jobs are active
  - Color-coded status badges
  - Job type labels
  
  Job States:
  - queued: Job is waiting to start (gray)
  - running: Job is currently executing (blue)
  - completed: Job finished successfully (green)
  - failed: Job encountered an error (red)
  - cancelled: Job was cancelled by user (yellow)
-->

<template>
  <!-- 
    Floating panel in bottom-right corner
    Only visible when there are active jobs (not completed/cancelled)
  -->
  <div 
    v-if="activeJobs.length > 0" 
    class="fixed bottom-4 right-4 bg-white shadow-2xl rounded-lg p-4 max-w-sm border border-gray-200 z-50"
  >
    <!-- Header -->
    <div class="flex justify-between items-center mb-3">
      <h3 class="font-semibold text-gray-800 flex items-center">
        <span class="mr-2">⚙️</span>
        Background Jobs
      </h3>
      <span class="text-xs text-gray-500">{{ activeJobs.length }} active</span>
    </div>

    <!-- Job List -->
    <div class="space-y-2">
      <div 
        v-for="job in activeJobs" 
        :key="job.id" 
        class="p-3 bg-gray-50 rounded border border-gray-200"
      >
        <!-- Job Header: Type and Status -->
        <div class="flex justify-between items-center mb-2">
          <span class="text-sm font-medium text-gray-700">
            {{ getJobLabel(job.type) }}
          </span>
          <span 
            :class="getStatusClass(job.status)" 
            class="text-xs px-2 py-1 rounded font-medium"
          >
            {{ getStatusLabel(job.status) }}
          </span>
        </div>

        <!-- Progress Bar (only for running jobs) -->
        <div v-if="job.status === 'running'" class="mb-2">
          <div class="w-full bg-gray-200 rounded-full h-2">
            <div 
              class="bg-blue-500 h-2 rounded-full transition-all duration-300" 
              :style="{ width: job.progress + '%' }"
            ></div>
          </div>
          <div class="text-xs text-gray-500 mt-1 text-right">
            {{ Math.round(job.progress) }}%
          </div>
        </div>

        <!-- Error Message (for failed jobs) -->
        <div v-if="job.status === 'failed' && job.error" class="mb-2">
          <p class="text-xs text-red-600 bg-red-50 p-2 rounded">
            {{ job.error }}
          </p>
        </div>

        <!-- Action Buttons -->
        <div class="flex justify-end gap-2">
          <!-- Cancel Button (only for queued/running jobs) -->
          <button 
            v-if="job.status === 'running' || job.status === 'queued'" 
            @click="handleCancelJob(job.id)"
            class="text-xs text-red-600 hover:text-red-800 hover:bg-red-50 px-2 py-1 rounded transition-colors"
          >
            ✕ Cancel
          </button>

          <!-- Dismiss Button (for completed/failed jobs) -->
          <button 
            v-if="job.status === 'completed' || job.status === 'failed'" 
            @click="handleDismissJob(job.id)"
            class="text-xs text-gray-600 hover:text-gray-800 hover:bg-gray-100 px-2 py-1 rounded transition-colors"
          >
            Dismiss
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * JobQueueStatus Component Script
 * 
 * Manages job queue state and handles Wails events for real-time updates.
 */

import { ref, computed, onMounted, onUnmounted } from 'vue';

// Import Wails runtime for events and backend calls
// These will be available when running in Wails environment
const EventsOn = window.runtime?.EventsOn;
const EventsOff = window.runtime?.EventsOff;
const CancelJob = window.go?.main?.App?.CancelJob;

// ============================================================================
// Component State
// ============================================================================

/** All jobs from the backend (includes completed/cancelled) */
const allJobs = ref([]);

/** Event listener cleanup function */
let cleanupEventListener = null;

// ============================================================================
// Computed Properties
// ============================================================================

/**
 * Active jobs (queued, running, or recently completed/failed)
 * Filters out old completed/cancelled jobs to keep the UI clean
 */
const activeJobs = computed(() => {
  return allJobs.value.filter(job => {
    // Always show queued and running jobs
    if (job.status === 'queued' || job.status === 'running') {
      return true;
    }

    // Show completed/failed/cancelled jobs for 5 seconds
    if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
      const completedAt = new Date(job.completedAt);
      const now = new Date();
      const ageInSeconds = (now - completedAt) / 1000;
      return ageInSeconds < 5; // Show for 5 seconds after completion
    }

    return false;
  });
});

// ============================================================================
// Lifecycle Hooks
// ============================================================================

/**
 * Component mounted - set up event listeners
 */
onMounted(() => {
  if (!EventsOn) {
    console.warn('Wails runtime not available, JobQueueStatus will not receive updates');
    return;
  }

  // Listen for job queue updates from backend
  cleanupEventListener = EventsOn('jobQueueUpdated', (updatedJobs) => {
    allJobs.value = updatedJobs || [];
  });
});

/**
 * Component unmounted - clean up event listeners
 */
onUnmounted(() => {
  if (cleanupEventListener && EventsOff) {
    EventsOff('jobQueueUpdated');
  }
});

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Get human-readable label for job type
 * 
 * @param {string} type - Job type (context_generation, diff_splitting, llm_call)
 * @returns {string} Human-readable label
 */
function getJobLabel(type) {
  const labels = {
    'context_generation': 'Generating Context',
    'diff_splitting': 'Splitting Diff',
    'llm_call': 'Calling AI',
  };
  return labels[type] || type;
}

/**
 * Get human-readable label for job status
 * 
 * @param {string} status - Job status (queued, running, completed, failed, cancelled)
 * @returns {string} Human-readable label
 */
function getStatusLabel(status) {
  const labels = {
    'queued': 'Queued',
    'running': 'Running',
    'completed': 'Done',
    'failed': 'Failed',
    'cancelled': 'Cancelled',
  };
  return labels[status] || status;
}

/**
 * Get CSS classes for status badge based on job status
 * 
 * @param {string} status - Job status
 * @returns {string} Tailwind CSS classes
 */
function getStatusClass(status) {
  const classes = {
    'queued': 'bg-gray-200 text-gray-700',
    'running': 'bg-blue-100 text-blue-700',
    'completed': 'bg-green-100 text-green-700',
    'failed': 'bg-red-100 text-red-700',
    'cancelled': 'bg-yellow-100 text-yellow-700',
  };
  return classes[status] || 'bg-gray-200 text-gray-700';
}

// ============================================================================
// Event Handlers
// ============================================================================

/**
 * Handle cancel job button click
 * Calls backend to cancel the job
 * 
 * @param {string} jobID - Unique identifier of the job to cancel
 */
async function handleCancelJob(jobID) {
  if (!CancelJob) {
    console.error('CancelJob method not available');
    return;
  }

  try {
    await CancelJob(jobID);
    console.log(`Cancelled job: ${jobID}`);
  } catch (error) {
    console.error(`Failed to cancel job ${jobID}:`, error);
  }
}

/**
 * Handle dismiss job button click
 * Removes the job from the active jobs list
 * 
 * @param {string} jobID - Unique identifier of the job to dismiss
 */
function handleDismissJob(jobID) {
  // Remove job from local state
  allJobs.value = allJobs.value.filter(job => job.id !== jobID);
}
</script>

<style scoped>
/**
 * Component-specific styles
 * Using scoped styles to avoid conflicts with global styles
 */

/* Smooth animations for progress bars */
.transition-all {
  transition-property: all;
  transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
  transition-duration: 300ms;
}

/* Ensure the floating panel is above other content */
.z-50 {
  z-index: 50;
}
</style>

