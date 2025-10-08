<template>
  <div class="flex items-center justify-center h-full bg-gradient-to-br from-blue-50 to-indigo-100">
    <div class="text-center max-w-2xl p-8">
      <div class="flex items-center justify-center mb-4">
        <img src="/appicon.png" alt="Shotgun Code" class="w-20 h-20 mr-4" />
      </div>
      <p class="text-ms text-gray-600 mb-8">
        One‑click codebase "blast" for Large‑Language‑Model workflows.
      </p>

      <div class="bg-white rounded-lg shadow-lg p-6 mb-8 text-left">
        <h2 class="text-lg font-semibold mb-4">How it works:</h2>
        <ol class="space-y-3">
          <li class="flex items-start">
            <span class="font-bold text-blue-600 mr-3">1.</span>
            <span>Select your project folder</span>
          </li>
          <li class="flex items-start">
            <span class="font-bold text-blue-600 mr-3">2.</span>
            <span>Choose files to include (with smart filtering)</span>
          </li>
          <li class="flex items-start">
            <span class="font-bold text-blue-600 mr-3">3.</span>
            <span>Generate unlimited context (no size limit)</span>
          </li>
          <li class="flex items-start">
            <span class="font-bold text-blue-600 mr-3">4.</span>
            <span>Get AI-generated diffs, intelligently split</span>
          </li>
        </ol>
      </div>

      <button
        @click="handleOpenFolder"
        class="px-8 py-4 bg-blue-600 text-white text-lg rounded-lg hover:bg-blue-700 shadow-lg transition-colors"
      >
        Open Project Folder
      </button>

      <!-- Attribution -->
      <div class="mt-8 text-xs text-gray-500 space-y-1">
        <p>
          Created by
          <a href="https://github.com/glebkudr" target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:underline">
            Gleb Curly
          </a>
        </p>
        <p>
          Redesigned by
          <a href="https://skpassegna.me" target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:underline">
            Samuel Kpassegna
          </a>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { navigateTo } from '../router';

/**
 * Handle opening the folder selection screen
 * Navigates to the folder selection screen where users can choose their project directory
 */
function handleOpenFolder() {
  navigateTo('folder');
}
</script>

