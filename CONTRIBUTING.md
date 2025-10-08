# Contributing to Shotgun Code

Thank you for your interest in contributing to Shotgun Code! This document provides guidelines and instructions for contributing to the project.

---

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [Getting Started](#getting-started)
3. [Development Setup](#development-setup)
4. [Project Structure](#project-structure)
5. [Development Workflow](#development-workflow)
6. [Coding Standards](#coding-standards)
7. [Testing](#testing)
8. [Submitting Changes](#submitting-changes)
9. [Reporting Bugs](#reporting-bugs)
10. [Feature Requests](#feature-requests)

---

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors. Please:

- Be respectful and constructive in discussions
- Welcome newcomers and help them get started
- Focus on what is best for the community
- Show empathy towards other community members

---

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go ≥ 1.20** - [Download Go](https://golang.org/dl/)
- **Node.js LTS** - [Download Node.js](https://nodejs.org/)
- **Wails CLI** - Install with: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Git** - [Download Git](https://git-scm.com/)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/shotgun_code.git
   cd shotgun_code
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/glebkudr/shotgun_code.git
   ```

---

## Development Setup

### 1. Install Dependencies

**Backend (Go):**
```bash
go mod tidy
```

**Frontend (Vue.js):**
```bash
cd frontend
npm install
cd ..
```

### 2. Run in Development Mode

```bash
wails dev
```

This will:
- Start the Go backend with hot reload
- Launch the Vue.js frontend with Vite dev server
- Open the application window

**Note:** Vue.js changes hot-reload automatically. For Go changes, restart the `wails dev` command.

### 3. Build for Production

```bash
wails build
```

Binaries will be created in `build/bin/`

---

## Project Structure

```
shotgun_code/
├── app.go              # Main application logic
├── job_queue.go        # Background job queue system
├── llm_client.go       # LLM API integration
├── split_diff.go       # Diff splitting algorithm
├── main.go             # Application entry point
├── go.mod              # Go dependencies
├── wails.json          # Wails configuration
├── ignore.glob         # Custom ignore patterns
├── frontend/           # Vue.js frontend
│   ├── src/
│   │   ├── components/ # Reusable Vue components
│   │   ├── screens/    # Screen components (9 screens)
│   │   ├── stores/     # Pinia state management
│   │   ├── composables/# Vue composables (keyboard, toast)
│   │   ├── router.js   # Custom screen-based router
│   │   └── main.js     # Frontend entry point
│   ├── package.json    # Node.js dependencies
│   └── vite.config.js  # Vite configuration
└── .github/
    └── workflows/
        └── build.yml   # CI/CD workflow
```

---

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

Use descriptive branch names:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test additions/updates

### 2. Make Your Changes

- Write clean, readable code
- Follow the coding standards (see below)
- Add comments for complex logic
- Update documentation as needed

### 3. Test Your Changes

- Test the application thoroughly
- Ensure no regressions in existing functionality
- Add tests for new features (when applicable)

### 4. Commit Your Changes

Use clear, descriptive commit messages:

```bash
git add .
git commit -m "feat: add keyboard shortcut for file search"
```

**Commit Message Format:**
```
<type>: <subject>

<body (optional)>

<footer (optional)>
```

**Types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Test additions/updates
- `chore:` - Build process or auxiliary tool changes

### 5. Keep Your Branch Updated

```bash
git fetch upstream
git rebase upstream/main
```

### 6. Push Your Changes

```bash
git push origin feature/your-feature-name
```

---

## Coding Standards

### Go (Backend)

- **Format code** with `go fmt` before committing
- **Follow** [Effective Go](https://golang.org/doc/effective_go) guidelines
- **Use meaningful variable names** - avoid single-letter names except for loops
- **Add comments** for exported functions and complex logic
- **Handle errors** explicitly - don't ignore errors
- **Use context** for cancellation and timeouts

**Example:**
```go
// GenerateContext creates a codebase snapshot from selected files
// Returns the context string and any error encountered
func (a *App) GenerateContext(selectedFiles []string) (string, error) {
    if len(selectedFiles) == 0 {
        return "", fmt.Errorf("no files selected")
    }
    // ... implementation
}
```

### Vue.js (Frontend)

- **Use Vue 3 Composition API** - prefer `<script setup>` syntax
- **Follow** [Vue.js Style Guide](https://vuejs.org/style-guide/)
- **Use Tailwind CSS** for styling - avoid custom CSS when possible
- **Component naming** - use PascalCase for component files
- **Props validation** - always define prop types
- **Emit events** - use descriptive event names

**Example:**
```vue
<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  files: {
    type: Array,
    required: true
  }
})

const emit = defineEmits(['file-selected'])

const selectedFile = ref(null)

const handleSelect = (file) => {
  selectedFile.value = file
  emit('file-selected', file)
}
</script>
```

### General Guidelines

- **Keep functions small** - one function, one responsibility
- **Avoid deep nesting** - extract complex logic into separate functions
- **Use constants** for magic numbers and strings
- **Write self-documenting code** - code should be readable without comments
- **Add comments** for "why", not "what"

---

## Testing

### Manual Testing

1. Test all 9 screens in the workflow
2. Test keyboard shortcuts
3. Test file selection and filtering
4. Test context generation with various file sizes
5. Test LLM API integration (if you have API keys)
6. Test diff splitting with large diffs
7. Test on different operating systems (Windows, macOS, Linux)

### Automated Testing

Currently, the project relies on manual testing. Contributions to add automated tests are welcome!

**Future testing areas:**
- Unit tests for Go functions
- Component tests for Vue components
- Integration tests for the full workflow
- E2E tests with Wails

---

## Submitting Changes

### Pull Request Process

1. **Ensure your code follows the coding standards**
2. **Update documentation** if you've changed functionality
3. **Test thoroughly** on your local machine
4. **Create a pull request** from your fork to the main repository
5. **Fill out the PR template** with all required information
6. **Wait for review** - maintainers will review your PR
7. **Address feedback** - make requested changes if needed
8. **Celebrate!** 🎉 Your contribution will be merged

### Pull Request Template

When creating a PR, include:

```markdown
## Description
Brief description of what this PR does

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
How did you test this change?

## Screenshots (if applicable)
Add screenshots for UI changes

## Checklist
- [ ] Code follows the project's coding standards
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings or errors
- [ ] Tested on multiple platforms (if applicable)
```

---

## Reporting Bugs

### Before Reporting

1. **Check existing issues** - your bug might already be reported
2. **Test on the latest version** - the bug might be fixed
3. **Gather information** - collect logs, screenshots, and steps to reproduce

### Bug Report Template

Create a new issue with:

```markdown
## Bug Description
Clear description of the bug

## Steps to Reproduce
1. Go to '...'
2. Click on '...'
3. See error

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Environment
- OS: [e.g., Windows 11, macOS 14, Ubuntu 22.04]
- Shotgun Code Version: [e.g., v1.0.0]
- Go Version: [e.g., 1.21]
- Node Version: [e.g., 20.10.0]

## Screenshots
If applicable, add screenshots

## Additional Context
Any other relevant information
```

---

## Feature Requests

We welcome feature requests! Please:

1. **Check existing issues** - your feature might already be requested
2. **Describe the problem** - what problem does this feature solve?
3. **Propose a solution** - how should this feature work?
4. **Consider alternatives** - are there other ways to solve this?

### Feature Request Template

```markdown
## Feature Description
Clear description of the feature

## Problem Statement
What problem does this solve?

## Proposed Solution
How should this feature work?

## Alternatives Considered
Other ways to solve this problem

## Additional Context
Any other relevant information
```

---

## Questions?

If you have questions about contributing:

- **Open a discussion** on GitHub Discussions
- **Ask in an issue** if it's related to a specific issue
- **Check the README** for general information

---

## License

By contributing to Shotgun Code, you agree that your contributions will be licensed under the same license as the project (see [LICENSE.md](LICENSE.md)).

---

**Thank you for contributing to Shotgun Code!** 🚀

Your contributions help make this tool better for everyone.

