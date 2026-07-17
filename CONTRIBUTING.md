# Contributing to OKF-go

First off, thank you for taking the time to contribute! 🎉

This document outlines the guidelines and workflow for contributing to OKF-go. Following these instructions helps ensure a smooth, efficient process for everyone involved.

---

## 🛠️ Getting Started

### Prerequisites

* Go `1.26` or higher

### Building the CLI Tool

Clone the repository and compile the binary:

```bash
# Build the binary locally
go build -o okf main.go

# Verify installation
./okf help
```

---

## 🛠️ Setting Up Your Development Environment

### Prerequisites

* **Go**: Version `1.26` or higher.
* **Task**: The [Taskfile](https://taskfile.dev) runner is used to manage development workflows.
  * On macOS: `brew install go-task`
  * On Linux: `sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d`

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

The project utilizes [Task](https://taskfile.dev) for local development and build orchestration. If you have `task` installed, you can use the following commands:

* **Build locally:** `task build` (compiles and outputs `./okf` with build metadata)
* **Run unit tests:** `task test`
* **Run lint checks:** `task lint`
* **Local cross-compilation:** `task cross-compile` (outputs multi-platform binaries to `dist/bin/`)
* **Clean build directory:** `task clean`

If you don't have `task` installed, you can build manually via:

```bash
go build -o okf main.go
```

Below is a detailed breakdown of the common commands.

### Vetting and Formatting

Before committing any changes, run the linter and formatter to check for stylistic and structural issues:

```bash
task lint
```

*Note: This runs `go vet` and checks that the code conforms to standard Go formatting rules.*

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

## 🧪 Testing the Codebase

Unit tests cover the parsing, validation, and assembly logic. Run the test suite:

```bash
go test ./... -v
```

If you have `task` installed, you can also run:

```bash
task test
```

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

## 🚀 Automated Releasing

Releases are built, packaged, and published to GitHub automatically using [GoReleaser](https://goreleaser.com) and GitHub Actions.

### How to trigger a new release

1. Ensure all changes are committed and pushed to the `main` branch.
2. Create and push a new semantic version tag:

   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

3. The **Release** GitHub Action workflow (`.github/workflows/release.yml`) will automatically run, compile the code for all major target operating systems/architectures, package them into archives, and publish a new GitHub Release with release notes and checksums.

---

## 📈 Benchmarks

OKF ships micro-benchmarks for its core hot paths. Please ensure you do not introduce performance regressions when contributing to the following packages: `parser`, `assembly`, `validator`.

```bash
task benchmark
```

For a full guide on interpreting results, detecting regressions with `benchstat`, and profiling, see [PERFORMANCE.md](PERFORMANCE.md).

---

## 📝 Code Guidelines

* **Idiomatic Go**: Follow standard Go design patterns and idioms (see [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)).
* **Documentation**: Write clear comments for exported structs, interfaces, and functions.
* **Testing**: Every new feature or bug fix should be accompanied by relevant unit tests. Keep test coverage high.
