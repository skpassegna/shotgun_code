<template>
  <ul class="enhanced-file-tree">
    <!-- 
      Iterate through each file/folder node in the tree
      Each node represents either a file or directory with metadata
    -->
    <li 
      v-for="node in filteredNodes" 
      :key="node.path" 
      :class="{ 
        'excluded-node': node.excluded,
        'search-match': isSearchMatch(node)
      }"
    >
      <!-- Node item container with indentation based on depth -->
      <div 
        class="node-item" 
        :style="{ 'padding-left': depth * 20 + 'px' }"
        :title="getNodeTooltip(node)"
      >
        <!-- 
          Expand/collapse toggle for directories
          Only shown for folders, not files
        -->
        <span 
          v-if="node.isDir" 
          @click="toggleExpand(node)" 
          class="toggler cursor-pointer select-none"
        >
          {{ node.expanded ? '▼' : '▶' }}
        </span>
        <!-- Spacer for files to align with folders -->
        <span v-else class="item-spacer inline-block w-4"></span>
        
        <!-- 
          Tri-state checkbox for inclusion/exclusion
          - Checked: File/folder is included
          - Unchecked: File/folder is excluded
          - Indeterminate: Folder has mixed children (some included, some excluded)
        -->
        <input 
          type="checkbox" 
          :checked="!node.excluded" 
          :indeterminate="node.isDir && isPartiallySelected(node)"
          @change="handleCheckboxChange(node)"
          class="exclude-checkbox mr-2 cursor-pointer"
          :disabled="isParentExcluded(node)"
        />
        
        <!-- 
          File/folder icon based on type and extension
          Provides visual cues for different file types
        -->
        <span class="node-icon mr-1">{{ getIcon(node) }}</span>
        
        <!-- 
          File/folder name with click handler
          Clicking folder name toggles expansion
          Search matches are highlighted
        -->
        <span 
          @click="node.isDir ? toggleExpand(node) : null" 
          :class="{ 
            'folder-name': node.isDir,
            'cursor-pointer': node.isDir,
            'font-semibold': node.isDir
          }"
          v-html="highlightSearchMatch(node.name)"
        ></span>
        
        <!--
          File size display (only for files, not directories)
          Shows human-readable size (KB, MB)
        -->
        <span
          v-if="!node.isDir && node.size"
          class="text-xs text-gray-500 ml-2"
        >
          {{ formatFileSize(node.size) }}
        </span>

        <!--
          Binary file indicator
          Shows a warning badge for binary files
        -->
        <span
          v-if="!node.isDir && node.isBinary"
          class="badge badge-binary ml-2"
          title="Binary file - will be skipped in context generation"
        >
          🔒 binary
        </span>

        <!--
          Exclusion badges showing why a file/folder is excluded
          - .gitignore: Matched by .gitignore rules
          - custom: Matched by custom ignore patterns
          - manual: Manually excluded by user
        -->
        <span v-if="node.isGitignored" class="badge badge-gitignore ml-2">
          .gitignore
        </span>
        <span v-if="node.isCustomIgnored" class="badge badge-custom ml-2">
          custom
        </span>
        <span v-if="node.manuallyExcluded" class="badge badge-manual ml-2">
          manual
        </span>
      </div>
      
      <!-- 
        Recursive rendering of child nodes
        Only shown when:
        1. Node is a directory
        2. Directory is expanded
        3. Directory has children
      -->
      <EnhancedFileTree
        v-if="node.isDir && node.expanded && node.children && node.children.length > 0"
        :nodes="node.children"
        :depth="depth + 1"
        :search-query="searchQuery"
        @toggle-exclude="$emit('toggle-exclude', $event)"
        @toggle-expand="$emit('toggle-expand', $event)"
      />
    </li>
  </ul>
</template>

<script setup>
import { computed } from 'vue';

/**
 * EnhancedFileTree Component
 * 
 * A recursive file tree component with advanced features:
 * - Tri-state checkboxes (checked, unchecked, indeterminate)
 * - File type icons
 * - File size display
 * - Exclusion badges (.gitignore, custom, manual)
 * - Search highlighting
 * - Expand/collapse functionality
 * 
 * Props:
 * @param {Array} nodes - Array of FileNode objects to display
 * @param {Number} depth - Current nesting depth (for indentation)
 * @param {String} searchQuery - Search term to filter and highlight nodes
 * 
 * Emits:
 * @event toggle-exclude - When a checkbox is clicked (node)
 * @event toggle-expand - When a folder is expanded/collapsed (node)
 */

const props = defineProps({
  nodes: {
    type: Array,
    required: true,
    default: () => []
  },
  depth: {
    type: Number,
    default: 0
  },
  searchQuery: {
    type: String,
    default: ''
  }
});

const emit = defineEmits(['toggle-exclude', 'toggle-expand']);

/**
 * Filtered nodes based on search query
 * If search query is empty, show all nodes
 * Otherwise, show only nodes that match the search or have matching descendants
 */
const filteredNodes = computed(() => {
  if (!props.searchQuery || props.searchQuery.trim() === '') {
    return props.nodes;
  }
  
  return props.nodes.filter(node => {
    // Check if current node matches
    if (nodeMatchesSearch(node)) {
      return true;
    }
    
    // Check if any descendant matches (for directories)
    if (node.isDir && node.children) {
      return hasMatchingDescendant(node);
    }
    
    return false;
  });
});

/**
 * Check if a node matches the search query
 * Case-insensitive search on node name
 */
function nodeMatchesSearch(node) {
  const query = props.searchQuery.toLowerCase();
  return node.name.toLowerCase().includes(query);
}

/**
 * Recursively check if a directory has any descendant matching the search
 */
function hasMatchingDescendant(node) {
  if (!node.children) return false;
  
  for (const child of node.children) {
    if (nodeMatchesSearch(child)) {
      return true;
    }
    if (child.isDir && hasMatchingDescendant(child)) {
      return true;
    }
  }
  
  return false;
}

/**
 * Check if a node is a search match
 * Used for highlighting matching nodes
 */
function isSearchMatch(node) {
  return props.searchQuery && nodeMatchesSearch(node);
}

/**
 * Highlight search matches in node name
 * Wraps matching text in <mark> tags for highlighting
 */
function highlightSearchMatch(name) {
  if (!props.searchQuery || props.searchQuery.trim() === '') {
    return name;
  }
  
  const query = props.searchQuery;
  const regex = new RegExp(`(${escapeRegex(query)})`, 'gi');
  return name.replace(regex, '<mark class="bg-yellow-200">$1</mark>');
}

/**
 * Escape special regex characters in search query
 */
function escapeRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Toggle expand/collapse state of a directory node
 */
function toggleExpand(node) {
  if (node.isDir) {
    node.expanded = !node.expanded;
    emit('toggle-expand', node);
  }
}

/**
 * Handle checkbox change event
 * Emits toggle-exclude event to parent component
 */
function handleCheckboxChange(node) {
  emit('toggle-exclude', node);
}

/**
 * Check if a directory is partially selected
 * Returns true if some (but not all) children are selected
 * Used for indeterminate checkbox state
 */
function isPartiallySelected(node) {
  if (!node.isDir || !node.children || node.children.length === 0) {
    return false;
  }
  
  const excludedCount = node.children.filter(child => child.excluded).length;
  const totalCount = node.children.length;
  
  // Partially selected if some (but not all) children are excluded
  return excludedCount > 0 && excludedCount < totalCount;
}

/**
 * Check if any parent of this node is excluded
 * If parent is excluded, this node should be disabled
 */
function isParentExcluded(node) {
  // This would require parent references in the node structure
  // For now, return false - can be enhanced later
  return false;
}

/**
 * Get appropriate icon for a file or folder
 * Based on file extension or directory status
 */
function getIcon(node) {
  if (node.isDir) {
    return node.expanded ? '📂' : '📁';
  }
  
  // Get file extension
  const ext = node.name.split('.').pop()?.toLowerCase();
  
  // Icon mapping for common file types
  const iconMap = {
    // Web
    'vue': '🖼️',
    'html': '🌐',
    'css': '🎨',
    'scss': '🎨',
    'sass': '🎨',
    
    // JavaScript/TypeScript
    'js': '📜',
    'jsx': '⚛️',
    'ts': '📘',
    'tsx': '⚛️',
    'json': '⚙️',
    
    // Backend
    'go': '🐹',
    'py': '🐍',
    'java': '☕',
    'rb': '💎',
    'php': '🐘',
    
    // Documentation
    'md': '📝',
    'txt': '📄',
    'pdf': '📕',
    
    // Images
    'png': '🖼️',
    'jpg': '🖼️',
    'jpeg': '🖼️',
    'gif': '🖼️',
    'svg': '🎨',
    'ico': '🖼️',
    
    // Config
    'yaml': '⚙️',
    'yml': '⚙️',
    'toml': '⚙️',
    'xml': '⚙️',
    'env': '🔐',
    
    // Build
    'sh': '🔧',
    'bash': '🔧',
    'dockerfile': '🐳',
  };
  
  return iconMap[ext] || '📄';
}

/**
 * Format file size in human-readable format
 * Converts bytes to KB, MB, etc.
 */
function formatFileSize(bytes) {
  if (!bytes || bytes === 0) return '0 B';
  
  const units = ['B', 'KB', 'MB', 'GB'];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${units[i]}`;
}

/**
 * Get tooltip text for a node
 * Shows full path and exclusion reason
 */
function getNodeTooltip(node) {
  let tooltip = node.path || node.relPath || node.name;
  
  if (node.excluded) {
    const reasons = [];
    if (node.isGitignored) reasons.push('.gitignore');
    if (node.isCustomIgnored) reasons.push('custom ignore');
    if (node.manuallyExcluded) reasons.push('manually excluded');
    
    if (reasons.length > 0) {
      tooltip += `\nExcluded by: ${reasons.join(', ')}`;
    }
  }
  
  return tooltip;
}
</script>

<style scoped>
/**
 * Enhanced File Tree Styles
 * Clean, modern styling with proper spacing and visual hierarchy
 */

.enhanced-file-tree {
  @apply list-none p-0 m-0;
}

.node-item {
  @apply flex items-center py-1 px-2 hover:bg-gray-50 rounded transition-colors;
}

/* Excluded nodes are grayed out */
.excluded-node .node-item {
  @apply opacity-50;
}

/* Search matches are highlighted */
.search-match > .node-item {
  @apply bg-yellow-50;
}

/* Folder names are clickable */
.folder-name {
  @apply select-none;
}

.folder-name:hover {
  @apply text-blue-600;
}

/* Exclusion badges */
.badge {
  @apply inline-block px-2 py-0.5 text-xs rounded-full font-medium;
}

.badge-gitignore {
  @apply bg-gray-200 text-gray-700;
}

.badge-custom {
  @apply bg-blue-200 text-blue-700;
}

.badge-manual {
  @apply bg-orange-200 text-orange-700;
}

.badge-binary {
  @apply bg-purple-100 text-purple-700 border border-purple-300;
}

/* Checkbox styling */
.exclude-checkbox {
  @apply cursor-pointer;
}

.exclude-checkbox:disabled {
  @apply cursor-not-allowed opacity-50;
}

/* Toggler (expand/collapse arrow) */
.toggler {
  @apply inline-block w-4 text-center text-gray-600;
}

.toggler:hover {
  @apply text-gray-900;
}
</style>

