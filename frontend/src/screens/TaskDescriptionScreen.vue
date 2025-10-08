<template>
  <div class="screen-container p-6 max-w-6xl mx-auto">
    <h1 class="text-3xl font-bold mb-6">Describe Your Task</h1>
    
    <div class="bg-white rounded-lg shadow p-6 mb-6">
      <p class="text-gray-600 mb-4">
        Describe what you want the AI to do. Be as specific as possible.
      </p>
      
      <textarea 
        v-model="taskDescription"
        placeholder="Example: Add a user authentication system with JWT tokens, including login, logout, and password reset functionality..."
        class="w-full h-48 px-4 py-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
      ></textarea>
      
      <div class="mt-4">
        <button 
          @click="showCustomRules = !showCustomRules"
          class="text-sm text-blue-600 hover:underline"
        >
          {{ showCustomRules ? '− Hide' : '+ Add' }} custom rules
        </button>
        
        <div v-if="showCustomRules" class="mt-3">
          <textarea 
            v-model="customRules"
            placeholder="Add any custom rules or constraints (e.g., 'Use TypeScript', 'Follow MVC pattern', etc.)"
            class="w-full h-32 px-4 py-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none text-sm"
          ></textarea>
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
        v-if="taskDescription.trim()"
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

// Get store
const store = useAppStore();

// Use store values
const taskDescription = computed({
  get: () => store.taskDescription,
  set: (value) => store.setTaskDescription(value)
});

const customRules = computed({
  get: () => store.customRules,
  set: (value) => store.setCustomRules(value)
});

const showCustomRules = ref(false);

function handleBack() {
  navigateBack();
}

function handleNext() {
  if (taskDescription.value.trim()) {
    navigateTo('review');
  }
}
</script>

