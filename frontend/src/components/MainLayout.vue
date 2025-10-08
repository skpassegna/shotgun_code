<template>
  <div class="h-screen flex flex-col">
    <!-- Breadcrumb Navigation (hidden on welcome screen) -->
    <nav v-if="currentScreen !== 'welcome'" class="bg-gray-100 px-4 py-2 border-b border-gray-300">
      <div class="flex items-center gap-2 text-sm">
        <span
          v-for="(crumb, i) in breadcrumbs"
          :key="i"
          class="flex items-center"
        >
          <button
            @click="navigateToBreadcrumb(crumb.screen)"
            class="text-blue-600 hover:underline"
            :class="{ 'font-semibold': i === breadcrumbs.length - 1 }"
          >
            {{ crumb.label }}
          </button>
          <span v-if="i < breadcrumbs.length - 1" class="mx-2 text-gray-400">›</span>
        </span>
      </div>
    </nav>

    <!-- Screen Router -->
    <main class="flex-1 overflow-auto bg-white">
      <WelcomeScreen v-if="currentScreen === 'welcome'" />
      <FolderSelectionScreen v-else-if="currentScreen === 'folder'" />
      <FileSelectionScreen v-else-if="currentScreen === 'files'" />
      <ModeSelectionScreen v-else-if="currentScreen === 'mode'" />
      <TaskDescriptionScreen v-else-if="currentScreen === 'task'" />
      <PromptReviewScreen v-else-if="currentScreen === 'review'" />
      <ExecutionScreen v-else-if="currentScreen === 'execute'" />
      <SplitDiffScreen v-else-if="currentScreen === 'split'" />
      <ApplyPatchScreen v-else-if="currentScreen === 'apply'" />
    </main>

    <!-- Job Queue Status (floating panel) -->
    <JobQueueStatus />

    <!-- Toast Notifications -->
    <ToastContainer />

    <!-- Status Bar -->
    <footer class="bg-gray-800 text-white px-4 py-1 text-xs flex justify-between items-center">
      <div class="flex items-center gap-4">
        <span>{{ statusMessage }}</span>
        <button
          @click="showAboutModal = true"
          class="text-gray-300 hover:text-white transition-colors flex items-center gap-1"
          title="About Shotgun Code"
        >
          <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
          </svg>
          About
        </button>
      </div>
      <span>{{ projectRoot || 'No project selected' }}</span>
    </footer>

    <!-- About Modal -->
    <AboutModal :is-open="showAboutModal" @close="showAboutModal = false" />
  </div>
</template>

<script setup>
/**
 * MainLayout Component
 *
 * The main application layout that manages screen routing and displays the job queue status.
 * This component uses a simple screen-based router (no Vue Router needed) and shows
 * breadcrumb navigation for easy navigation between screens.
 *
 * Features:
 * - Screen-based routing with breadcrumb navigation
 * - Job queue status display (floating panel)
 * - Status bar with project information
 * - About modal with attribution
 * - Clean, minimal layout
 */

import { ref } from 'vue';
import { currentScreen, breadcrumbs, navigateToBreadcrumb } from '../router';

// Import all screen components
import WelcomeScreen from '../screens/WelcomeScreen.vue';
import FolderSelectionScreen from '../screens/FolderSelectionScreen.vue';
import FileSelectionScreen from '../screens/FileSelectionScreen.vue';
import ModeSelectionScreen from '../screens/ModeSelectionScreen.vue';
import TaskDescriptionScreen from '../screens/TaskDescriptionScreen.vue';
import PromptReviewScreen from '../screens/PromptReviewScreen.vue';
import ExecutionScreen from '../screens/ExecutionScreen.vue';
import SplitDiffScreen from '../screens/SplitDiffScreen.vue';
import ApplyPatchScreen from '../screens/ApplyPatchScreen.vue';

// Import job queue status component
import JobQueueStatus from './JobQueueStatus.vue';
import ToastContainer from './ToastContainer.vue';
import AboutModal from './AboutModal.vue';

// Import global store
import { useAppStore } from '../stores/appStore';
import { computed } from 'vue';

// Get store instance
const store = useAppStore();

// Application state
const statusMessage = ref('Ready');
const projectRoot = computed(() => store.projectFolder || 'No project selected');
const showAboutModal = ref(false);

</script>

<style scoped>
.flex-1 {
  min-height: 0;
}
</style> 