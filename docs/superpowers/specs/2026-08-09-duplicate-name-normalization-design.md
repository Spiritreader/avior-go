# Duplicate Name Normalization Design

## Goal

Treat media files as duplicates when their names differ only by whitespace or hyphen/underscore separators, without renaming or rewriting the original files.

## Required behavior

The duplicate comparison key must normalize both the incoming candidate and existing library names:

- remove all whitespace;
- remove all hyphens (`-`) and underscores (`_`);
- compare case-insensitively;
- compare the normalized basename and preserve extension handling so configured media extensions remain part of the duplicate contract.

Examples that must compare equal:

```text
aktiv und gesund - Faszientherapie - Poolkeime - Stand-up-Paddling.mkv
aktiv und gesund   Faszientherapie   Poolkeime   Stand-up-Paddling.mkv
aktiv und gesund _ Faszientherapie _ Poolkeime _ Stand-up-Paddling.mkv
```

The internal hyphen in `Stand-up-Paddling` is also removed, as requested. The normalization applies only to comparison keys; stored paths, displayed names, output names, and cache entries remain unchanged.

## Integration points

Apply the same helper to both duplicate lookup paths:

- fresh filesystem scanning;
- in-memory library-cache scanning.

The year-aware duplicate lookup automatically benefits because it uses the same duplicate scan.

## Non-goals

- Do not rename existing files.
- Do not change output naming.
- Do not broadly remove punctuation other than whitespace, hyphens, and underscores.
- Do not alter duplicate module behavior after a match is found.

## Verification

Add deterministic tests covering:

1. all three supplied names normalize to the same key;
2. internal hyphens are removed;
3. case and whitespace differences compare equal;
4. unrelated names remain different;
5. the original name remains unchanged after key generation.
