/**
 * Shotgun Code - Global Application Store
 * 
 * This Pinia store manages all global application state that needs to be shared
 * between different screens in the workflow.
 * 
 * State Management:
 * - Project folder selection
 * - File tree and selected files
 * - Mode selection (dev, architect, debug, tasks)
 * - Task description and custom rules
 * - Generated context and prompt
 * - LLM provider settings
 * - Generated diff and splits
 * 
 * This eliminates the need for prop drilling and allows any component
 * to access or modify the application state.
 */

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export const useAppStore = defineStore('app', () => {
  // ============================================================================
  // State - Step 1: Folder Selection
  // ============================================================================
  
  /** Selected project folder path */
  const projectFolder = ref('');
  
  // ============================================================================
  // State - Step 2: File Selection
  // ============================================================================
  
  /** Complete file tree from backend */
  const fileTree = ref([]);
  
  /** Map of file paths to their selection state */
  const selectedFiles = ref(new Map());
  
  /** Map of file paths to their exclusion state */
  const excludedFiles = ref(new Map());
  
  // ============================================================================
  // State - Step 3: Mode Selection
  // ============================================================================
  
  /** Selected mode: 'dev', 'architect', 'debug', or 'tasks' */
  const selectedMode = ref('');
  
  // ============================================================================
  // State - Step 4: Task Description
  // ============================================================================
  
  /** User's task description */
  const taskDescription = ref('');
  
  /** Custom rules/constraints */
  const customRules = ref('');
  
  // ============================================================================
  // State - Step 5: Context Generation & Prompt Review
  // ============================================================================
  
  /** Generated codebase context */
  const generatedContext = ref('');
  
  /** Final composed prompt */
  const composedPrompt = ref('');
  
  /** Token count estimation */
  const estimatedTokens = ref(0);
  
  /** Cost estimation */
  const estimatedCost = ref(0);
  
  // ============================================================================
  // State - Step 6: LLM Execution
  // ============================================================================
  
  /** Selected LLM provider */
  const llmProvider = ref('google');
  
  /** LLM model name */
  const llmModel = ref('');
  
  /** API key for the selected provider */
  const apiKey = ref('');
  
  /** Custom API base URL (for custom provider) */
  const customBaseURL = ref('');
  
  /** Temperature setting */
  const temperature = ref(0.7);
  
  /** Max tokens to generate */
  const maxTokens = ref(4096);
  
  /** LLM response (generated diff) */
  const llmResponse = ref('');
  
  // ============================================================================
  // State - Step 7: Diff Splitting
  // ============================================================================
  
  /** Lines per split configuration */
  const linesPerSplit = ref(500);
  
  /** Array of split diffs */
  const splitDiffs = ref([]);
  
  // ============================================================================
  // State - General
  // ============================================================================
  
  /** Loading state */
  const isLoading = ref(false);
  
  /** Error message */
  const errorMessage = ref('');
  
  /** Success message */
  const successMessage = ref('');
  
  // ============================================================================
  // Computed Properties
  // ============================================================================
  
  /**
   * Get list of selected file paths
   */
  const selectedFilePaths = computed(() => {
    const paths = [];
    for (const [path, isSelected] of selectedFiles.value.entries()) {
      if (isSelected) {
        paths.push(path);
      }
    }
    return paths;
  });
  
  /**
   * Get list of excluded file paths
   */
  const excludedFilePaths = computed(() => {
    const paths = [];
    for (const [path, isExcluded] of excludedFiles.value.entries()) {
      if (isExcluded) {
        paths.push(path);
      }
    }
    return paths;
  });
  
  /**
   * Count of selected files
   */
  const selectedFileCount = computed(() => {
    return selectedFilePaths.value.length;
  });
  
  /**
   * Check if workflow can proceed to next step
   */
  const canProceedToFileSelection = computed(() => {
    return projectFolder.value !== '';
  });
  
  const canProceedToModeSelection = computed(() => {
    return selectedFileCount.value > 0;
  });
  
  const canProceedToTaskDescription = computed(() => {
    return selectedMode.value !== '';
  });
  
  const canProceedToPromptReview = computed(() => {
    return taskDescription.value.trim() !== '';
  });
  
  const canProceedToExecution = computed(() => {
    return composedPrompt.value !== '';
  });
  
  // ============================================================================
  // Actions - Folder Selection
  // ============================================================================
  
  /**
   * Set the selected project folder
   */
  function setProjectFolder(path) {
    if (path && typeof path === 'string') {
      projectFolder.value = path.trim();
    } else {
      console.error('Invalid folder path:', path);
      projectFolder.value = '';
    }
  }

  /**
   * Clear project folder
   */
  function clearProjectFolder() {
    projectFolder.value = '';
  }
  
  // ============================================================================
  // Actions - File Selection
  // ============================================================================
  
  /**
   * Set the file tree
   */
  function setFileTree(tree) {
    if (Array.isArray(tree)) {
      fileTree.value = tree;
    } else {
      console.error('Invalid file tree (must be array):', tree);
      fileTree.value = [];
    }
  }

  /**
   * Toggle file selection
   */
  function toggleFileSelection(path, isSelected) {
    if (path && typeof path === 'string' && typeof isSelected === 'boolean') {
      if (!(selectedFiles.value instanceof Map)) {
        selectedFiles.value = new Map();
      }
      selectedFiles.value.set(path, isSelected);
    } else {
      console.error('Invalid toggleFileSelection params:', { path, isSelected });
    }
  }

  /**
   * Toggle file exclusion
   */
  function toggleFileExclusion(path, isExcluded) {
    if (path && typeof path === 'string' && typeof isExcluded === 'boolean') {
      if (!(excludedFiles.value instanceof Map)) {
        excludedFiles.value = new Map();
      }
      excludedFiles.value.set(path, isExcluded);
    } else {
      console.error('Invalid toggleFileExclusion params:', { path, isExcluded });
    }
  }

  /**
   * Clear all file selections
   */
  function clearFileSelections() {
    if (selectedFiles.value instanceof Map) {
      selectedFiles.value.clear();
    } else {
      selectedFiles.value = new Map();
    }
    if (excludedFiles.value instanceof Map) {
      excludedFiles.value.clear();
    } else {
      excludedFiles.value = new Map();
    }
  }
  
  // ============================================================================
  // Actions - Mode Selection
  // ============================================================================
  
  /**
   * Set the selected mode
   */
  function setMode(mode) {
    const validModes = ['dev', 'architect', 'debug', 'tasks'];
    if (mode && typeof mode === 'string' && validModes.includes(mode)) {
      selectedMode.value = mode;
    } else {
      console.error('Invalid mode (must be dev/architect/debug/tasks):', mode);
      selectedMode.value = '';
    }
  }
  
  // ============================================================================
  // Actions - Task Description
  // ============================================================================
  
  /**
   * Set task description
   */
  function setTaskDescription(description) {
    if (typeof description === 'string') {
      taskDescription.value = description;
    } else {
      console.error('Invalid task description (must be string):', description);
      taskDescription.value = '';
    }
  }

  /**
   * Set custom rules
   */
  function setCustomRules(rules) {
    if (typeof rules === 'string') {
      customRules.value = rules;
    } else {
      console.error('Invalid custom rules (must be string):', rules);
      customRules.value = '';
    }
  }
  
  // ============================================================================
  // Actions - Context & Prompt
  // ============================================================================
  
  /**
   * Set generated context
   */
  function setGeneratedContext(context) {
    if (typeof context === 'string') {
      generatedContext.value = context;
    } else {
      console.error('Invalid context (must be string):', context);
      generatedContext.value = '';
    }
  }

  /**
   * Set composed prompt
   */
  function setComposedPrompt(prompt) {
    if (typeof prompt === 'string') {
      composedPrompt.value = prompt;
    } else {
      console.error('Invalid prompt (must be string):', prompt);
      composedPrompt.value = '';
    }
  }

  /**
   * Set token estimation
   */
  function setEstimatedTokens(tokens) {
    const numTokens = Number(tokens);
    if (!isNaN(numTokens) && numTokens >= 0) {
      estimatedTokens.value = Math.floor(numTokens);
    } else {
      console.error('Invalid token count (must be non-negative number):', tokens);
      estimatedTokens.value = 0;
    }
  }

  /**
   * Set cost estimation
   */
  function setEstimatedCost(cost) {
    const numCost = Number(cost);
    if (!isNaN(numCost) && numCost >= 0) {
      estimatedCost.value = numCost;
    } else {
      console.error('Invalid cost (must be non-negative number):', cost);
      estimatedCost.value = 0;
    }
  }
  
  // ============================================================================
  // Actions - LLM Settings
  // ============================================================================
  
  /**
   * Set LLM provider
   */
  function setLLMProvider(provider) {
    const validProviders = ['google', 'openai', 'anthropic', 'custom'];
    if (provider && typeof provider === 'string' && validProviders.includes(provider)) {
      llmProvider.value = provider;
    } else {
      console.error('Invalid LLM provider (must be google/openai/anthropic/custom):', provider);
      llmProvider.value = 'google';
    }
  }

  /**
   * Set LLM model
   */
  function setLLMModel(model) {
    if (typeof model === 'string') {
      llmModel.value = model;
    } else {
      console.error('Invalid LLM model (must be string):', model);
      llmModel.value = '';
    }
  }

  /**
   * Set API key
   */
  function setAPIKey(key) {
    if (typeof key === 'string') {
      apiKey.value = key;
    } else {
      console.error('Invalid API key (must be string):', key);
      apiKey.value = '';
    }
  }

  /**
   * Set custom base URL
   */
  function setCustomBaseURL(url) {
    if (typeof url === 'string') {
      customBaseURL.value = url;
    } else {
      console.error('Invalid base URL (must be string):', url);
      customBaseURL.value = '';
    }
  }

  /**
   * Set temperature
   */
  function setTemperature(temp) {
    const numTemp = Number(temp);
    if (!isNaN(numTemp) && numTemp >= 0 && numTemp <= 2) {
      temperature.value = numTemp;
    } else {
      console.error('Invalid temperature (must be 0-2):', temp);
      temperature.value = 0.7;
    }
  }

  /**
   * Set max tokens
   */
  function setMaxTokens(tokens) {
    const numTokens = Number(tokens);
    if (!isNaN(numTokens) && numTokens > 0) {
      maxTokens.value = Math.floor(numTokens);
    } else {
      console.error('Invalid max tokens (must be positive number):', tokens);
      maxTokens.value = 4096;
    }
  }

  /**
   * Set LLM response
   */
  function setLLMResponse(response) {
    if (typeof response === 'string') {
      llmResponse.value = response;
    } else {
      console.error('Invalid LLM response (must be string):', response);
      llmResponse.value = '';
    }
  }
  
  // ============================================================================
  // Actions - Diff Splitting
  // ============================================================================
  
  /**
   * Set lines per split
   */
  function setLinesPerSplit(lines) {
    const numLines = Number(lines);
    if (!isNaN(numLines) && numLines >= 100 && numLines <= 10000) {
      linesPerSplit.value = Math.floor(numLines);
    } else {
      console.error('Invalid lines per split (must be 100-10000):', lines);
      linesPerSplit.value = 500;
    }
  }

  /**
   * Set split diffs
   */
  function setSplitDiffs(splits) {
    if (Array.isArray(splits)) {
      splitDiffs.value = splits;
    } else {
      console.error('Invalid split diffs (must be array):', splits);
      splitDiffs.value = [];
    }
  }
  
  // ============================================================================
  // Actions - UI State
  // ============================================================================
  
  /**
   * Set loading state
   */
  function setLoading(loading) {
    isLoading.value = loading;
  }
  
  /**
   * Set error message
   */
  function setError(message) {
    errorMessage.value = message;
  }
  
  /**
   * Clear error message
   */
  function clearError() {
    errorMessage.value = '';
  }
  
  /**
   * Set success message
   */
  function setSuccess(message) {
    successMessage.value = message;
  }
  
  /**
   * Clear success message
   */
  function clearSuccess() {
    successMessage.value = '';
  }
  
  /**
   * Reset entire store to initial state
   */
  function resetStore() {
    try {
      projectFolder.value = '';
      fileTree.value = [];

      // Safely clear Maps
      if (selectedFiles.value instanceof Map) {
        selectedFiles.value.clear();
      } else {
        selectedFiles.value = new Map();
      }

      if (excludedFiles.value instanceof Map) {
        excludedFiles.value.clear();
      } else {
        excludedFiles.value = new Map();
      }

      selectedMode.value = '';
      taskDescription.value = '';
      customRules.value = '';
      generatedContext.value = '';
      composedPrompt.value = '';
      estimatedTokens.value = 0;
      estimatedCost.value = 0;
      llmProvider.value = 'google';
      llmModel.value = '';
      apiKey.value = '';
      customBaseURL.value = '';
      temperature.value = 0.7;
      maxTokens.value = 4096;
      llmResponse.value = '';
      linesPerSplit.value = 500;
      splitDiffs.value = [];
      isLoading.value = false;
      errorMessage.value = '';
      successMessage.value = '';
    } catch (error) {
      console.error('Error resetting store:', error);
      // Force reset critical values even if error occurs
      projectFolder.value = '';
      selectedMode.value = '';
      llmProvider.value = 'google';
    }
  }
  
  // Return all state and actions
  return {
    // State
    projectFolder,
    fileTree,
    selectedFiles,
    excludedFiles,
    selectedMode,
    taskDescription,
    customRules,
    generatedContext,
    composedPrompt,
    estimatedTokens,
    estimatedCost,
    llmProvider,
    llmModel,
    apiKey,
    customBaseURL,
    temperature,
    maxTokens,
    llmResponse,
    linesPerSplit,
    splitDiffs,
    isLoading,
    errorMessage,
    successMessage,
    
    // Computed
    selectedFilePaths,
    excludedFilePaths,
    selectedFileCount,
    canProceedToFileSelection,
    canProceedToModeSelection,
    canProceedToTaskDescription,
    canProceedToPromptReview,
    canProceedToExecution,
    
    // Actions
    setProjectFolder,
    clearProjectFolder,
    setFileTree,
    toggleFileSelection,
    toggleFileExclusion,
    clearFileSelections,
    setMode,
    setTaskDescription,
    setCustomRules,
    setGeneratedContext,
    setComposedPrompt,
    setEstimatedTokens,
    setEstimatedCost,
    setLLMProvider,
    setLLMModel,
    setAPIKey,
    setCustomBaseURL,
    setTemperature,
    setMaxTokens,
    setLLMResponse,
    setLinesPerSplit,
    setSplitDiffs,
    setLoading,
    setError,
    clearError,
    setSuccess,
    clearSuccess,
    resetStore,
  };
});

