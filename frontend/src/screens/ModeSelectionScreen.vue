<template>
  <div class="screen-container p-6 max-w-6xl mx-auto">
    <h1 class="text-3xl font-bold mb-6">Choose Mode</h1>
    
    <div class="bg-white rounded-lg shadow p-6 mb-6">
      <p class="text-gray-600 mb-6">
        What would you like the AI to help you with?
      </p>
      
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <button 
          @click="selectMode('dev')"
          class="p-6 border-2 rounded-lg text-left hover:border-blue-500 hover:bg-blue-50 transition-all"
          :class="selectedMode === 'dev' ? 'border-blue-500 bg-blue-50' : 'border-gray-200'"
        >
          <div class="text-2xl mb-2">💻</div>
          <h3 class="font-semibold text-lg mb-2">Generate Code</h3>
          <p class="text-sm text-gray-600">Write new features, refactor code, or implement functionality</p>
        </button>
        
        <button 
          @click="selectMode('architect')"
          class="p-6 border-2 rounded-lg text-left hover:border-blue-500 hover:bg-blue-50 transition-all"
          :class="selectedMode === 'architect' ? 'border-blue-500 bg-blue-50' : 'border-gray-200'"
        >
          <div class="text-2xl mb-2">🏗️</div>
          <h3 class="font-semibold text-lg mb-2">Architecture Plan</h3>
          <p class="text-sm text-gray-600">Design system architecture, plan refactoring, or review structure</p>
        </button>
        
        <button 
          @click="selectMode('debug')"
          class="p-6 border-2 rounded-lg text-left hover:border-blue-500 hover:bg-blue-50 transition-all"
          :class="selectedMode === 'debug' ? 'border-blue-500 bg-blue-50' : 'border-gray-200'"
        >
          <div class="text-2xl mb-2">🐛</div>
          <h3 class="font-semibold text-lg mb-2">Find Bugs</h3>
          <p class="text-sm text-gray-600">Identify issues, security vulnerabilities, or code smells</p>
        </button>
        
        <button 
          @click="selectMode('tasks')"
          class="p-6 border-2 rounded-lg text-left hover:border-blue-500 hover:bg-blue-50 transition-all"
          :class="selectedMode === 'tasks' ? 'border-blue-500 bg-blue-50' : 'border-gray-200'"
        >
          <div class="text-2xl mb-2">📋</div>
          <h3 class="font-semibold text-lg mb-2">Update Tasks</h3>
          <p class="text-sm text-gray-600">Generate task lists, update documentation, or plan sprints</p>
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
        v-if="selectedMode"
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

// Get store
const store = useAppStore();

// Get selected mode from store
const selectedMode = computed(() => store.selectedMode);

/**
 * Select a mode and save to store
 */
function selectMode(mode) {
  store.setMode(mode);
}

function handleBack() {
  navigateBack();
}

function handleNext() {
  if (selectedMode.value) {
    navigateTo('task');
  }
}
</script>

