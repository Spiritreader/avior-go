# Task 03: Extend CI Workflow to Windows+Linux

## Goal

`.github/workflows/go.yml` builds both binaries on release and attaches them to the
release. The existing `wangyoucao577/go-release-action@v1.18` step is parameterized
per OS (matrix) instead of introducing a second mechanism (preserving the existing
convention).

## Edit: Replace `.github/workflows/go.yml` entirely with

```yaml
name: Go

on:
  release:
    types: [created, edited]
    branches: [ master ]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        include:
          - goos: windows
            goarch: amd64
            binary_name: avior-go
          - goos: linux
            goarch: amd64
            binary_name: avior-go
    steps:
    - uses: actions/checkout@v2
    - uses: wangyoucao577/go-release-action@v1.18
      with:
        github_token: ${{ secrets.GITHUB_TOKEN }}
        goos: ${{ matrix.goos }}
        goarch: ${{ matrix.goarch }}
        binary_name: ${{ matrix.binary_name }}
        ldflags: -s -w
```

Reason: matrix instead of two hard jobs: same configuration, artifacts both land on the
same release (`go-release-action` attaches one asset per `goos`; Windows asset is
named `avior-go_windows_amd64.exe.zip`, Linux `avior-go_linux_amd64.tar.gz` —
naming convention of the action, verify on first release).

## Check

- Validate workflow YAML locally: `python -c "import yaml,sys; yaml.safe_load(open('.github/workflows/go.yml'))"`.
- Real check only possible on next release tag — alternative verification:
  simulate the matrix run locally with the same commands
  (`CGO_ENABLED=0 GOOS=linux go build ./...`, see Task 04).

## Dependency

Task 01 must be merged, otherwise the Linux target will fail in CI.
