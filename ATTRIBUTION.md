# Attribution & Credits

## About Shotgun Code

Shotgun Code is a desktop application that solves LLM context limits by generating massive codebase snapshots and intelligently splitting large diffs for AI coding assistants.

---

## Original Creator

### Gleb Curly

**Role:** Project Creator & Original Developer

Gleb Curly is the original creator and developer of Shotgun Code. He built the initial version of this tool, implementing the core functionality including:
- Go backend with Wails framework
- File system operations and context generation
- Diff splitting algorithm
- Original Vue.js frontend
- Cross-platform desktop application architecture

**Connect with Gleb:**
- 🔗 GitHub: [@glebkudr](https://github.com/glebkudr)
- 🔗 LinkedIn: [glebkudr](https://www.linkedin.com/in/glebkudr/)
- 🔗 X (Twitter): [@glebcurly](https://x.com/glebcurly)
- 📦 Original Repository: [shotgun_code](https://github.com/glebkudr/shotgun_code)

---

## UX/UI Redesign

### Samuel Kpassegna

**Role:** UX/UI Redesign & Enhancement

Samuel Kpassegna redesigned the user experience and interface, transforming Shotgun Code into a modern, VSCode-style application with enhanced usability:

**Key Contributions:**
- Complete UX/UI redesign with VSCode-inspired interface
- Multi-screen onboarding workflow
- Enhanced file tree with search and filtering
- Modern component architecture (Vue 3 + Tailwind CSS)
- Improved navigation with breadcrumbs
- Toast notification system
- Job queue status tracking
- Keyboard shortcuts
- About modal with attribution
- Responsive and accessible design

**Connect with Samuel:**
- 🌐 Website: [skpassegna.me](https://skpassegna.me)
- 🔗 GitHub: [skpassegna.link/github](https://skpassegna.link/github)
- 🔗 LinkedIn: [skpassegna.link/linkedin](https://skpassegna.link/linkedin)
- 🔗 X (Twitter): [skpassegna.link/twitter](https://skpassegna.link/twitter)
- 🔗 Facebook: [skpassegna.link/facebook](https://skpassegna.link/facebook)

---

## Technology Stack

### Backend
- **Go** - Core application logic
- **Wails v2** - Desktop application framework
- **fsnotify** - File system watching
- **Advanced bin-packing algorithm** - Diff splitting

### Frontend
- **Vue.js 3** - Progressive JavaScript framework
- **Pinia** - State management
- **Tailwind CSS** - Utility-first CSS framework
- **Vite** - Build tool and dev server

### Features
- Asynchronous context generation with progress tracking
- File watching with hot reload
- Intelligent diff splitting
- Multi-tier clipboard fallback (WSL→Wails→Browser)
- Reactive UI with debounced updates
- Custom screen-based router
- Toast notifications
- Keyboard shortcuts

---

## License

Custom MIT-like license - see `LICENSE.md` file for details.

---

## Contributing

Contributions are welcome! Please:
- Format Go code with `go fmt`
- Follow Vue 3 style guidelines
- Maintain the existing code structure
- Add tests for new features
- Update documentation as needed

---

## Acknowledgments

Special thanks to:
- The Wails team for the excellent desktop framework
- The Vue.js community for the reactive framework
- The Tailwind CSS team for the utility-first CSS framework
- All contributors and users of Shotgun Code

---

**Shotgun Code** - Load, aim, blast your code straight into the mind of an LLM.

