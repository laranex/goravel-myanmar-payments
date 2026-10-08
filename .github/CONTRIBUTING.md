# Contribution Guide

Thank you for considering contributing to Goravel Myanmar Payments! Please review the following guidelines before submitting a pull request.

For significant changes, please open an issue first so we can discuss the approach.

## Process

1. Fork the project
2. Create a new branch
3. Code, test, commit, and push
4. Open a pull request detailing your changes

## Guidelines

- Ensure the code is formatted with `gofmt` and passes `go vet ./...`.
- This package is glue between Goravel and `github.com/laranex/go-myanmar-payments/v4`. Gateway rules (signing, validation, endpoints, statuses) belong in the SDK; never reimplement them here.
- Keep the behavior in line with `laranex/laravel-myanmar-payments` (config keys, environment variable names, the auto-submit form route).
- Send a coherent commit history, making sure each commit in your pull request is meaningful.
- You may need to [rebase](https://git-scm.com/book/en/v2/Git-Branching-Rebasing) to avoid merge conflicts.
- Please remember that we follow [SemVer](http://semver.org/); the import path carries the major version (`/v4`).

## Setup

Clone your fork; Go 1.25 or higher is the only requirement:

```bash
go version
```

To work against a local checkout of the SDK, add an uncommitted `go.work` (it is gitignored):

```bash
go work init .
go work edit -replace=github.com/laranex/go-myanmar-payments/v4=../go-myanmar-payments
```

## Lint

Format and vet your code:

```bash
gofmt -l .
go vet ./...
```

## Tests

Run all tests:

```bash
go test -race ./...
```
