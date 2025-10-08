<template>
  <div class="screen-container p-6 max-w-6xl mx-auto">
    <h1 class="text-3xl font-bold mb-6">Execute Prompt</h1>
    
    <div class="bg-white rounded-lg shadow p-6 mb-6">
      <p class="text-gray-600 mb-6">
        Choose how to execute your prompt:
      </p>
      
      <!-- Execution mode tabs -->
      <div class="flex gap-2 mb-6 border-b">
        <button 
          @click="executionMode = 'api'"
          class="px-4 py-2 font-semibold transition-colors"
          :class="executionMode === 'api' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-gray-500 hover:text-gray-700'"
        >
          Direct API Call
        </button>
        <button 
          @click="executionMode = 'manual'"
          class="px-4 py-2 font-semibold transition-colors"
          :class="executionMode === 'manual' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-gray-500 hover:text-gray-700'"
        >
          Manual Copy-Paste
        </button>
      </div>
      
      <!-- API mode -->
      <div v-if="executionMode === 'api'" class="space-y-4">
        <div>
          <label class="block text-sm font-semibold mb-2">Provider:</label>
          <select
            v-model="provider"
            class="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="google">Google AI Studio (Gemini)</option>
            <option value="openai">OpenAI (GPT-4/GPT-5)</option>
            <option value="anthropic">Anthropic (Claude)</option>
            <option value="custom">Custom OpenAI-Compatible API</option>
          </select>
        </div>

        <div v-if="provider === 'custom'">
          <label class="block text-sm font-semibold mb-2">Base URL:</label>
          <input
            v-model="baseURL"
            type="text"
            placeholder="http://localhost:8080"
            class="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div>
          <label class="block text-sm font-semibold mb-2">Model:</label>
          <input
            v-model="model"
            type="text"
            :placeholder="getModelPlaceholder()"
            class="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p class="text-xs text-gray-500 mt-1">{{ getModelHint() }}</p>
        </div>

        <div>
          <label class="block text-sm font-semibold mb-2">API Key:</label>
          <input
            v-model="apiKey"
            type="password"
            placeholder="Enter your API key..."
            class="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-semibold mb-2">Temperature:</label>
            <input
              v-model.number="temperature"
              type="number"
              min="0"
              max="1"
              step="0.1"
              class="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label class="block text-sm font-semibold mb-2">Max Tokens:</label>
            <input
              v-model.number="maxTokens"
              type="number"
              min="100"
              max="100000"
              step="100"
              class="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        <button
          @click="executeWithAPI"
          :disabled="isExecuting || !apiKey"
          class="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <span v-if="isExecuting">⏳ Executing...</span>
          <span v-else>🚀 Execute with API</span>
        </button>
      </div>
      
      <!-- Manual mode -->
      <div v-else class="space-y-4">
        <div class="p-4 bg-blue-50 rounded border border-blue-200">
          <p class="text-sm text-gray-700">
            1. The prompt has been copied to your clipboard<br>
            2. Paste it into your preferred LLM (Google AI Studio, ChatGPT, Claude, etc.)<br>
            3. Copy the generated diff and paste it below
          </p>
        </div>

        <textarea
          v-model="manualDiff"
          placeholder="Paste the generated git diff here..."
          class="w-full h-48 px-4 py-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none font-mono text-sm"
        ></textarea>

        <button
          @click="saveManualDiff"
          :disabled="!manualDiff.trim()"
          class="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Save Diff
        </button>
      </div>

      <!-- Response area (shown after execution) -->
      <div v-if="llmResponse" class="mt-6">
        <label class="block text-sm font-semibold mb-2">AI Response:</label>
        <textarea
          v-model="llmResponse"
          readonly
          class="w-full h-64 px-4 py-3 border rounded-lg bg-gray-50 font-mono text-sm resize-none"
          placeholder="AI response will appear here..."
        ></textarea>

        <button
          @click="copyResponse"
          class="mt-2 px-4 py-2 bg-gray-100 text-gray-700 rounded hover:bg-gray-200 transition-colors"
        >
          📋 Copy Response
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
        class="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
      >
        Next →
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import { navigateTo, navigateBack } from '../router';
import { useAppStore } from '../stores/appStore';
import { useToast } from '../composables/useToast';

// Get Wails backend methods
const CallLLMAPI = window.go?.main?.App?.CallLLMAPI;

// Get store and toast
const store = useAppStore();
const { showSuccess, showError, showInfo } = useToast();

// Local state
const executionMode = ref('api');
const isExecuting = ref(false);
const manualDiff = ref('');

// Form state (synced with store)
const provider = computed({
  get: () => store.llmProvider,
  set: (value) => store.setLLMProvider(value)
});

const model = computed({
  get: () => store.llmModel,
  set: (value) => store.setLLMModel(value)
});

const apiKey = computed({
  get: () => store.apiKey,
  set: (value) => store.setAPIKey(value)
});

const baseURL = computed({
  get: () => store.customBaseURL,
  set: (value) => store.setCustomBaseURL(value)
});

const temperature = computed({
  get: () => store.temperature,
  set: (value) => store.setTemperature(value)
});

const maxTokens = computed({
  get: () => store.maxTokens,
  set: (value) => store.setMaxTokens(value)
});

const llmResponse = computed(() => store.llmResponse);

/**
 * Get model placeholder based on provider
 */
function getModelPlaceholder() {
  const placeholders = {
    google: 'gemini-2.5-flash',
    openai: 'gpt-5-mini',
    anthropic: 'claude-sonnet-4-5-20250929',
    custom: 'your-model-name'
  };
  return placeholders[provider.value] || '';
}

/**
 * Get model hint based on provider
 */
function getModelHint() {
  const hints = {
    google: 'e.g., gemini-2.5-flash, gemini-2.5-pro',
    openai: 'e.g., gpt-5, gpt-5-mini, gpt-5-nano',
    anthropic: 'e.g., claude-sonnet-4-5-20250929',
    custom: 'Specify the model name for your custom API'
  };
  return hints[provider.value] || '';
}

/**
 * Execute prompt with LLM API
 */
async function executeWithAPI() {
  if (!CallLLMAPI) {
    showError('Backend not available. Please ensure the application is running properly.');
    return;
  }

  if (!apiKey.value) {
    showError('Please enter your API key.');
    return;
  }

  if (!store.composedPrompt) {
    showError('No prompt available. Please go back and generate a prompt.');
    return;
  }

  isExecuting.value = true;
  showInfo('Calling LLM API... This may take a minute.');

  try {
    // Set up event listener for LLM response
    const cleanup = window.runtime?.EventsOn('llmResponseReceived', (response) => {
      store.setLLMResponse(response.content);
      isExecuting.value = false;
      showSuccess('LLM response received successfully!');

      // Cleanup event listener
      if (cleanup && window.runtime?.EventsOff) {
        window.runtime.EventsOff('llmResponseReceived');
      }
    });

    // Call LLM API (runs as background job)
    const jobID = await CallLLMAPI(
      provider.value,
      apiKey.value,
      store.composedPrompt,
      model.value || getModelPlaceholder(),
      temperature.value,
      maxTokens.value
    );

    console.log('LLM API call started with job ID:', jobID);

  } catch (error) {
    console.error('Error calling LLM API:', error);
    showError('Failed to call LLM API. Please try again.');
    isExecuting.value = false;
  }
}

/**
 * Save manually pasted diff
 */
function saveManualDiff() {
  if (!manualDiff.value.trim()) {
    showError('Please paste a diff before saving.');
    return;
  }

  store.setLLMResponse(manualDiff.value);
  showSuccess('Diff saved successfully!');
}

/**
 * Copy response to clipboard
 */
async function copyResponse() {
  try {
    await navigator.clipboard.writeText(llmResponse.value);
    showSuccess('Response copied to clipboard!');
  } catch (error) {
    console.error('Failed to copy:', error);
    showError('Failed to copy to clipboard.');
  }
}

function handleBack() {
  navigateBack();
}

function handleNext() {
  if (!llmResponse.value) {
    showError('Please execute the prompt or paste a diff before proceeding.');
    return;
  }
  navigateTo('split');
}

// Copy prompt to clipboard when entering manual mode
onMounted(async () => {
  if (executionMode.value === 'manual' && store.composedPrompt) {
    try {
      await navigator.clipboard.writeText(store.composedPrompt);
      showInfo('Prompt copied to clipboard! Paste it into your preferred LLM.');
    } catch (error) {
      console.error('Failed to copy prompt:', error);
    }
  }
});
</script>

