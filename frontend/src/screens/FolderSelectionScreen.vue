<template>
  <div class="screen-container p-6 max-w-6xl mx-auto">
    <h1 class="text-3xl font-bold mb-6">Select Project Folder</h1>
    
    <div class="bg-white rounded-lg shadow p-6 mb-6">
      <p class="text-gray-600 mb-4">
        Choose the root folder of your project. Shotgun Code will scan all files and directories.
      </p>
      
      <button 
        @click="handleSelectFolder" 
        class="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
      >
        📁 Browse for Folder
      </button>
      
      <div v-if="selectedFolder" class="mt-4 p-4 bg-gray-50 rounded border">
        <p class="text-sm text-gray-500 mb-1">Selected folder:</p>
        <p class="font-mono text-sm">{{ selectedFolder }}</p>
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
        v-if="selectedFolder"
        @click="handleNext" 
        class="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
      >
        Next →
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { navigateTo, navigateBack } from '../router';
import { useAppStore } from '../stores/appStore';
import { useToast } from '../composables/useToast';

// Get Wails backend method
const SelectDirectory = window.go?.main?.App?.SelectDirectory;

// Get store and toast
const store = useAppStore();
const { showSuccess, showError } = useToast();

// Computed property for selected folder from store
const selectedFolder = computed(() => store.projectFolder);

/**
 * Handle folder selection
 * Opens native directory picker and saves selection to store
 */
async function handleSelectFolder() {
  // Validate backend is available
  if (!SelectDirectory || typeof SelectDirectory !== 'function') {
    showError('Backend not available. Please ensure the application is running properly.');
    console.error('SelectDirectory method not available:', SelectDirectory);
    return;
  }

  try {
    // Call Wails backend to open directory picker
    const folderPath = await SelectDirectory();

    // Validate response
    if (!folderPath || typeof folderPath !== 'string' || folderPath.trim() === '') {
      // User cancelled the dialog or invalid response
      return;
    }

    // Save to store (store will validate)
    store.setProjectFolder(folderPath);

    // Verify it was saved
    if (store.projectFolder && store.projectFolder === folderPath.trim()) {
      showSuccess('Project folder selected successfully!');
    } else {
      showError('Failed to save folder selection.');
    }
  } catch (error) {
    console.error('Error selecting folder:', error);
    showError(`Failed to select folder: ${error?.message || 'Unknown error'}`);
  }
}

function handleBack() {
  navigateBack();
}

function handleNext() {
  // Validate folder is selected
  if (!selectedFolder.value || typeof selectedFolder.value !== 'string' || selectedFolder.value.trim() === '') {
    showError('Please select a project folder before proceeding.');
    return;
  }

  navigateTo('files');
}
</script>

