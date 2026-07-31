# Task 03: CI-Workflow auf Windows+Linux erweitern

## Ziel

`.github/workflows/go.yml` baut bei einem Release beide Binaries und hängt sie ans Release.
Der bestehende `wangyoucao577/go-release-action@v1.18`-Schritt wird pro OS parametrisiert
(Matrix), statt einen zweiten Mechanismus einzuführen (bestehende Konvention beibehalten).

## Edit: `.github/workflows/go.yml` komplett ersetzen durch

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

Begründung Matrix statt zwei harter Jobs: gleiche Konfiguration, Artefakte landen beide
am selben Release (`go-release-action` hängt pro `goos` ein Asset an; Windows-Asset heißt
`avior-go_windows_amd64.exe.zip` bzw. Linux `avior-go_linux_amd64.tar.gz` —
Namenskonvention der Action, [INFERENCE] aus der Action-Doku, beim ersten Release prüfen).

## Check

- Workflow-YAML lokal validieren: `python -c "import yaml,sys; yaml.safe_load(open('.github/workflows/go.yml'))"`.
- Echter Check erst beim nächsten Release-Tag möglich — Alternativ-Verifikation:
  den Matrix-Lauf lokal mit den gleichen Befehlen simulieren
  (`CGO_ENABLED=0 GOOS=linux go build ./...`, siehe Task 04).

## Abhängigkeit

Task 01 muss gemergt sein, sonst schlägt das Linux-Target in CI fehl.
