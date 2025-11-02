# Contributing to go-relax

Thanks for your interest in contributing to go-relax!

Table of contents
- How to contribute
- Reporting bugs
- Submitting changes (PR workflow)
- Testing and linting
- Commit message guidelines
- Code of conduct

How to contribute
1. Fork the repository.
2. Create a branch from master:
   - git checkout -b <your-branch-name>
3. Make small, focused changes with tests where appropriate.
4. Run the tests and linters locally (instructions below).
5. Push your branch to your fork and open a Pull Request against srfrog/go-relax master.

Reporting bugs
- Open an issue using the Bug Report template.
- Include: steps to reproduce, expected vs actual behavior, Go version, OS.

Submitting changes (PR workflow)
- Open a pull request from your fork to srfrog/go-relax master.
- Use the provided PR template and fill in all checklist items.
- Keep PRs focused and small where possible.
- If requested, respond to review comments and push updates to the same branch.
- If you allow “edits by maintainers” that helps maintainers make small fixes.

Testing and linting (local)
- Install Go (1.20+ recommended).
- Run unit tests:
  - go test ./...
- Run go vet:
  - go vet ./...
- Run gofmt check:
  - gofmt -l .   # should print no files
- Optional linting (recommended):
  - golangci-lint run

Commit message guidelines
- Keep a short subject line (<=72 chars).
- Use the imperative mood: "Fix", "Add", "Remove".
- Optionally include a longer body explaining why the change is necessary.
- If you must sign-off, include Signed-off-by: Name <email>.

Code of Conduct
- Be respectful and constructive in reviews and issues. Maintainers reserve the right to close or redirect contributions that do not follow community standards.