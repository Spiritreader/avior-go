# Description-Aware Year Collision LLM Comparison Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent false year suffixes by treating recordings with different years but equivalent descriptions as existing duplicates, using the shared `movie_nfo_lib` prompt and score pipeline for semantic comparison.

**Architecture:** Keep deterministic description normalization in Go for cheap obvious matches. For non-identical but sufficiently populated descriptions, avior-go invokes a small Python adapter that imports the private `movie_nfo_lib` package and calls its existing `smart_compare_cached` implementation. The adapter returns strict JSON containing the shared-library score and explanation; Go compares the score against the configurable `DescriptionSimilarityThreshold` (default `0.7`). The Docker runtime clones and installs the pinned shared-library revision at startup using `GIT_TOKEN` supplied by the instance environment; the token never enters the image or logs.

**Tech Stack:** Go 1.25, Python 3.10+, `movie_metadata.smart_compare_cached`, Ollama via the shared library, Docker Compose, JSON over stdin/stdout.

## Global Constraints

- Use the existing model-specific prompts and score parser from `movie_nfo_lib`; do not copy prompts into avior-go.
- `DescriptionSimilarityThreshold` is configurable in `config.json`, defaults to `0.7`, and must be finite and within `[0.0, 1.0]`.
- Different release years plus equivalent descriptions means existing duplicate; do not append a year suffix.
- Different descriptions or a missing description means the descriptions are not equal; retain year suffix behavior.
- Normalize punctuation, whitespace, Unicode, and recording-date decorations before the deterministic comparison.
- `GIT_TOKEN` is supplied by the Avior-go instance environment; never commit, bake, print, or log its value.
- Pin the shared-library Git revision/tag; do not clone arbitrary `HEAD`.
- LLM errors, malformed responses, missing configuration, and timeouts must be observable and must not silently report semantic equality.
- Preserve the existing `YearAwareDupes` switch and duplicate module behavior.
- Do not modify the central `movie_nfo_lib` repository from avior-go.
- Never bump `github.com/karrick/godirwalk` to v1.17.0.

---

## Feasibility Findings

The shared implementation can provide the planned request/response:

- `movie_metadata.llm.smart_compare_cached(nfo, txt, cfg)` returns `(score, explanation)`.
- `movie_metadata.llm.run_llm` selects `OLLAMA_BASE_URL`/`OLLAMA_MODEL`, supports the fallback instance, and uses the model-specific prompt resolved from `movie_metadata/llm_prompts.json`.
- The prompt files require `Score: <float>` in the range 0.0–1.0.
- `extract_score_and_expl` parses that score, and the shared cache keys on both input texts plus prompt/model configuration.
- The Python package requires Python >=3.10 and dependencies `requests`, `openai`, `PyYAML`, `httpx`, and `tiktoken`.
- The current avior-go image contains ffmpeg but no Python runtime, so the Docker runtime must add Python and the adapter dependencies.
- `GIT_TOKEN` alone does not make the package available: the entrypoint must clone/update the private repository before the adapter starts. The repository URL and pinned ref must be explicit configuration.

The planned protocol is therefore feasible, but only after adding Python/runtime provisioning and a startup dependency check. It cannot be implemented as a direct Go import.

---

## File Map

- Create: `llm_compare/compare.py` — JSON-lines adapter around `movie_metadata.llm.smart_compare_cached`.
- Create: `llm_compare/requirements.txt` — runtime adapter dependencies, pinned to compatible major versions or installed from the shared package metadata.
- Create: `llm_compare/README.md` — adapter protocol and local invocation.
- Modify: `media/media.go` — expose description extraction and normalized comparison input.
- Modify: `media/year.go` — add deterministic description normalization helpers and date-decoration removal.
- Test: `media/year_test.go` — normalization and description-equivalence cases.
- Modify: `worker/worker.go` — compare descriptions before appending release year; invoke adapter only for unresolved non-empty descriptions.
- Test: `worker/worker_test.go` — same-description existing decision, different-description suffix decision, missing-description behavior, adapter failure behavior.
- Modify: `config/config.go` — add `DescriptionSimilarityThreshold float64` with JSON persistence and default `0.7`, plus explicit adapter settings and pinned library ref defaults if runtime configuration is needed by Go.
- Modify: `Dockerfile` — add Python runtime/build dependencies without exposing `GIT_TOKEN` in image layers.
- Modify: `docker/entrypoint.sh` — clone/update the pinned library into `/data/.vendor/movie_nfo_lib`, validate it, and start avior-go with adapter environment.
- Modify: `compose.yaml` — forward `GIT_TOKEN` and non-secret library/adapter settings from the instance environment.
- Modify: `.dockerignore` — exclude local adapter caches and Python artifacts.
- Modify: `.github/workflows/go.yml` — build/test the Go adapter integration without requiring live Ollama; Docker integration remains a deployment smoke check.

---

## Task 1: Deterministic Description Extraction and Normalization

**Files:**
- Modify: `media/year.go`
- Modify: `media/media.go`
- Test: `media/year_test.go`

**Interfaces:**
- Produce `DescriptionFromMetadata(lines []string) string`.
- Produce `NormalizeDescription(description string) string`.
- Produce `DescriptionsEquivalent(left, right string) bool`.
- Empty descriptions are never equivalent.

- [ ] **Step 1: Write failing tests**

Cover:

```go
func TestDescriptionsEquivalentIgnoresRecordingDates(t *testing.T)
func TestDescriptionsEquivalentIgnoresPunctuationAndWhitespace(t *testing.T)
func TestDescriptionsEquivalentRequiresContent(t *testing.T)
func TestDescriptionFromMetadataUsesDescriptionLine(t *testing.T)
```

Use two descriptions with different recording dates and punctuation that must compare equal, and two descriptions with different plot content that must compare unequal.

- [ ] **Step 2: Run the focused tests and verify RED**

Run:

```text
go test ./media -run 'TestDescriptionsEquivalent|TestDescriptionFromMetadata' -count=1
```

Expected: FAIL because the new functions do not exist.

- [ ] **Step 3: Implement normalization**

Extract `Description=` only. Remove recording-date decorations such as `Erstsendung`, `Aufnahme`, `Sendedatum`, and date forms (`DD.MM.YYYY`, `YYYY-MM-DD`, `DD/MM/YYYY`) while preserving content-bearing years. Normalize Unicode case, punctuation to spaces, and repeated whitespace. Do not treat empty or date-only descriptions as equal.

- [ ] **Step 4: Run focused tests and verify GREEN**

Expected: all new media tests PASS.

- [ ] **Step 5: Commit**

```text
git add media/year.go media/media.go media/year_test.go
git commit -m "feat: normalize descriptions for year collision checks"
```

---

## Task 2: Python Adapter Using the Shared Library

**Files:**
- Create: `llm_compare/compare.py`
- Create: `llm_compare/requirements.txt`
- Create: `llm_compare/README.md`
- Test: `llm_compare/test_compare.py`

**Interfaces:**
- stdin: one JSON object per invocation:
  ```json
  {"left":"...", "right":"...", "threshold":0.7}
  ```
- stdout: exactly one JSON object:
  ```json
  {"score":0.83,"same_content":true,"explanation":"..."}
  ```
- error: non-zero exit and a concise stderr message; never include `GIT_TOKEN`.
- `same_content` is calculated as `score > threshold`, not `>=`; the adapter receives the validated threshold from avior-go and does not own the default.

**Configuration contract:**

```go
DescriptionSimilarityThreshold float64 `json:"DescriptionSimilarityThreshold"`
```

`config.InitWithDefaults` sets it to `0.7`. Values outside `[0.0, 1.0]` or non-finite values fail configuration validation rather than being silently clamped.
- [ ] **Step 1: Write failing adapter tests**

Test JSON parsing, default threshold `0.7`, custom thresholds such as `0.85`, bounds validation, strict `score > threshold`, malformed shared-library output, and missing descriptions. Mock only the shared-library call at the adapter boundary; do not mock JSON parsing.

- [ ] **Step 2: Run adapter tests and verify RED**

Run:

```text
python -m pytest llm_compare/test_compare.py -q
```

Expected: FAIL because the adapter does not exist.

- [ ] **Step 3: Implement the adapter**

Import `smart_compare_cached` from `movie_metadata.llm`; pass both descriptions as the two comparison texts. Receive the validated threshold from stdin, serialize score, threshold decision, and explanation, and reject non-finite or out-of-range scores and thresholds.

- [ ] **Step 4: Run adapter tests and verify GREEN**

Expected: all adapter tests PASS.

- [ ] **Step 5: Commit**

```text
git add llm_compare
git commit -m "feat: add shared-library description comparison adapter"
```

---

## Task 3: Worker Decision Integration

**Files:**
- Modify: `worker/worker.go`
- Test: `worker/worker_test.go`

**Interfaces:**
- Add an internal comparator boundary such as:
  ```go
  type DescriptionComparator interface {
      Compare(ctx context.Context, left, right string) (score float64, explanation string, err error)
  }
  ```
- The default implementation invokes the Python adapter with a bounded context and JSON stdin/stdout.
- Tests inject a deterministic comparator; production uses the adapter.

- [ ] **Step 1: Write failing worker tests**

Cover:

1. Existing duplicate has a different year but equivalent normalized description: no suffix; existing duplicate modules receive the original name.
2. Normalized descriptions differ and LLM score equals the configured threshold: append year.
3. LLM score just above the configured threshold: treat as existing; do not append year.
4. Missing description on either side: do not treat as equivalent; append year.
5. Adapter error: log the error and use the conservative non-equivalent path; append year.

Use real `media.File` metadata content and an injected comparator, not source-text assertions.

- [ ] **Step 2: Run tests and verify RED**

Run:

```text
go test ./worker -run 'Test.*Description|Test.*Year' -count=1
```

Expected: the new decision tests fail against the current year-only behavior.

- [ ] **Step 3: Implement the decision flow**

At the current year-collision branch:

1. Resolve the existing duplicate file's description.
2. If normalized descriptions are equal, treat it as existing and do not append a year.
3. If not equal and both descriptions are non-empty, call the adapter.
4. If score is greater than the configured `DescriptionSimilarityThreshold`, treat as existing and do not append a year.
5. Otherwise append the release year using existing `appendReleaseYear` behavior.
6. Log source (`normalized` or `llm`), score, and decision; never log full sensitive descriptions unnecessarily.

- [ ] **Step 4: Run worker tests and verify GREEN**

Expected: all new worker tests PASS and existing worker tests remain green.

- [ ] **Step 5: Commit**

```text
git add worker/worker.go worker/worker_test.go
git commit -m "feat: use description identity before year suffixes"
```

---

## Task 4: Docker Runtime and Instance Environment

**Files:**
- Modify: `Dockerfile`
- Modify: `docker/entrypoint.sh`
- Modify: `compose.yaml`
- Modify: `.dockerignore`
- Create: `docker/movie_nfo_lib.env.example`

**Interfaces:**
- Runtime variables:
  ```text
  GIT_TOKEN
  MOVIE_NFO_LIB_REPO
  MOVIE_NFO_LIB_REF
  DESCRIPTION_LLM_TIMEOUT
  OLLAMA_BASE_URL
  OLLAMA_MODEL
  OLLAMA_BASE_URL_FALLBACK
  OLLAMA_MODEL_FALLBACK
  OLLAMA_NUM_CTX
  ```
- `DescriptionSimilarityThreshold` is read from the persisted Avior config and passed per adapter request; an environment override must not silently override an explicit config value.
- Default repository ref is a fixed release tag/commit, never `HEAD`.

- [ ] **Step 1: Add configuration/startup tests or shell checks**

Test the entrypoint contract with a fake Git/Python environment: missing token, invalid checkout, successful checkout, and adapter import failure. Assert the token is not printed.

- [ ] **Step 2: Verify the current image cannot satisfy the contract**

Run the existing Dockerfile inspection/build check. Expected finding: runtime image has ffmpeg but no Python adapter/library installation.

- [ ] **Step 3: Implement runtime provisioning**

Add Python and the adapter dependencies to the runtime image. At startup, clone the pinned library into `/data/.vendor/movie_nfo_lib` using an authenticated Git URL without echoing it, install the package, validate `llm_prompts.json` and the selected model prompt, then launch avior-go. Forward instance environment variables through Compose; do not put secret values in YAML.

- [ ] **Step 4: Verify the Docker contract**

Run shell syntax checks and a non-secret configuration check. If Docker is available, build and run the image with a fake local Git repository and fake Ollama endpoint. In this workstation Docker is not assumed available; report that limitation rather than claiming a build.

- [ ] **Step 5: Commit**

```text
git add Dockerfile docker/entrypoint.sh compose.yaml .dockerignore docker/movie_nfo_lib.env.example
git commit -m "feat: provision shared metadata library for description comparison"
```

---

## Task 5: Prompt/Score Integration Verification and Documentation

**Files:**
- Modify: `README.md` or create `docs/description-year-collision.md`
- Modify: `media/realcase_test.go`
- Test: `llm_compare/test_compare.py`

- [ ] **Step 1: Add the ten real collision fixtures as metadata-only tests**

Represent the ten known cases with descriptions and expected outcomes. Do not require live Ollama in unit tests. Include the configured-threshold boundary, a custom threshold, and the default `0.7` boundary explicitly.

- [ ] **Step 2: Add an optional live integration command**

Document:

```text
python llm_compare/compare.py <<'JSON'
{"left":"...","right":"...","threshold":0.7}
JSON
```

The command must show the shared-library-selected model, score, and decision without printing `GIT_TOKEN`.

- [ ] **Step 3: Run the complete narrow verification**

Run:

```text
go test ./media ./worker
python -m pytest llm_compare -q
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...
```

Expected: all tests pass and both OS builds compile. Live semantic comparison is verified separately only when Ollama and the private library are available.

- [ ] **Step 4: Commit**

```text
git add docs media/realcase_test.go llm_compare
 git commit -m "test: verify description-aware year collision decisions"
```

---

## Acceptance Criteria

- The ten known collision examples are evaluated through description identity before a year suffix is appended.
- Same normalized description with different years is treated as existing and keeps the unsuffixed name.
- Different descriptions receive the year suffix.
- The configured threshold is persisted, validated, passed to the adapter, and used with strict `score > threshold` semantics; default `0.7` remains backward-compatible.
- The score comes from the shared library's selected prompt/model path, not a copied avior-go prompt.
- Fallback Ollama configuration and model-specific prompt selection remain available.
- Missing descriptions and adapter failures never produce a false semantic-equality result.
- `GIT_TOKEN` is provided by instance environment and never appears in source, image layers, stdout, or logs.
- Windows and Linux Go builds remain compilable.
