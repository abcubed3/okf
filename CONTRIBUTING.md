# Contributing to OKF-go

First off, thank you for taking the time to contribute! 🎉

This document outlines the guidelines and workflow for contributing to OKF-go. Following these instructions helps ensure a smooth, efficient process for everyone involved.

---

## 🛠️ Setting Up Your Development Environment

### Prerequisites
- **Go**: Version `1.26` or higher.
- **Task**: The [Taskfile](https://taskfile.dev) runner is used to manage development workflows.
  - On macOS: `brew install go-task`
  - On Linux: `sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d`

### Local Setup
1. Fork the repository on GitHub.
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/okf.git
   cd okf
   ```
3. Initialize/verify Go dependencies:
   ```bash
   go mod download
   ```

---

## 💻 Developer Workflow

The project uses `Taskfile` to simplify common development tasks.

### Vetting and Formatting
Before committing any changes, run the linter and formatter to check for stylistic and structural issues:
```bash
task lint
```
*Note: This runs `go vet` and checks that the code conforms to standard Go formatting rules.*

### Running Tests
Ensure all existing tests pass and add new tests for your modifications:
```bash
task test
```

### Local Compilation
Verify that the codebase compiles cleanly for your host operating system:
```bash
task build
```
This outputs the `okf` executable to the project root with the version set as `dev`.

### Local Cross-Compilation Check
If making changes to build or packaging systems, run:
```bash
task cross-compile
```
This builds binaries for all target architectures inside `dist/bin/`.

---

## 📬 Submitting a Pull Request

1. **Create a Branch**: Create a new feature branch for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```
2. **Commit Changes**: Write clear, concise commit messages. If your change fixes an open issue, reference it (e.g., `fixes #123`).
3. **Push & Open PR**: Push your branch to GitHub and open a Pull Request (PR) against the `main` branch of the root repository.
4. **Code Review**: A maintainer will review your pull request. Please address any review comments or requested changes.

---

## 📈 Benchmarks

OKF ships micro-benchmarks for its core hot paths. Please ensure you do not introduce performance regressions when contributing to the following packages: `parser`, `assembly`, `validator`.

```bash
task benchmark
```

For a full guide on interpreting results, detecting regressions with `benchstat`, and profiling, see [PERFORMANCE.md](PERFORMANCE.md).

---

## 📝 Code Guidelines

- **Idiomatic Go**: Follow standard Go design patterns and idioms (see [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)).
- **Documentation**: Write clear comments for exported structs, interfaces, and functions.
- **Testing**: Every new feature or bug fix should be accompanied by relevant unit tests. Keep test coverage high.
