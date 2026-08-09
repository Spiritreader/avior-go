# Duplicate Name Normalization Design

## Goal

Treat media files as duplicates when their names differ only by whitespace or hyphen/underscore separators, without renaming or rewriting the original files.

## Required behavior

The duplicate comparison key must normalize both the incoming candidate and existing library names:

- retain only Unicode letters and Unicode digits;
- remove all whitespace, hyphens, underscores, punctuation, and other symbols;
- compare case-insensitively;
- compare the normalized basename while preserving extension handling so configured media extensions remain part of the duplicate contract.

Examples that must compare equal:

```text
aktiv und gesund - Faszientherapie - Poolkeime - Stand-up-Paddling.mkv
aktiv und gesund   Faszientherapie   Poolkeime   Stand-up-Paddling.mkv
aktiv und gesund _ Faszientherapie _ Poolkeime _ Stand-up-Paddling.mkv
```

The internal hyphen in `Stand-up-Paddling` is also removed, as requested. The normalization applies only to comparison keys; stored paths, displayed names, output names, and cache entries remain unchanged. This intentionally accepts the small risk that genuinely different titles such as `A/B` and `AB` normalize identically.

## Integration points

Apply the same helper to both duplicate lookup paths:

- fresh filesystem scanning;
- in-memory library-cache scanning.

The year-aware duplicate lookup automatically benefits because it uses the same duplicate scan.

## Non-goals

- Do not rename existing files.
- Do not change output naming.
- Do not remove or rewrite characters from stored filenames; broad character removal is limited to comparison keys.
- Do not alter duplicate module behavior after a match is found.

## Verification

Add deterministic tests covering:

1. all three supplied names normalize to the same key;
2. internal hyphens and other punctuation are removed;
3. case and whitespace differences compare equal;
4. Unicode letters and digits are preserved;
5. unrelated names remain different where their retained letters/digits differ;
6. the original name remains unchanged after key generation.
