# CoPre (Contextual Presence) Anchoring Logic

This Go package provides functionality to find and score "anchors" within a text document. An anchor represents an occurrence of a specific search text (`searchText`) within a larger body of text (`oldText`).

## Purpose

The core function, `findAndScoreAnchors` (tested in `pkg/copre/anchoring_test.go`), is designed to locate alternative positions where a given piece of text (`searchText`) exists within a larger text (`oldText`), excluding its original specified position (`originalChangeStartPos`). This is particularly useful in scenarios where text has been modified, and we need to find the *most likely* corresponding location of a piece of text from the original version in the modified version. Examples include applying patches, synchronizing edits in collaborative environments, or refactoring code.

## How it Works

1.  **Input:**
    *   `oldText`: The original, potentially multi-line text content.
    *   `searchText`: The specific string to search for within `oldText`.
    *   `originalChangeStartPos`: The starting byte position (0-indexed) of the instance of `searchText` that should *not* be considered as an anchor. This represents the "original" location of the text before some change occurred.

2.  **Finding Candidates:** The function searches `oldText` for all occurrences of `searchText`.

3.  **Filtering:** It filters out the occurrence that starts exactly at `originalChangeStartPos`.

4.  **Context Extraction:** For the instance at `originalChangeStartPos` and for each remaining candidate anchor, the function extracts the text immediately preceding (prefix) and immediately following (affix) the `searchText`.

5.  **Scoring:** Each candidate anchor is assigned a score based on how well its prefix and affix match the prefix and affix of the original occurrence at `originalChangeStartPos`.
    *   A base score is awarded for finding an occurrence.
    *   Additional points are added for matching prefixes and affixes. The amount added depends on the length and similarity of the matching context. Longer and more exact context matches result in higher scores.
    *   The goal is to find anchors whose surrounding context is most similar to the original context, making them strong candidates for being the "same" piece of text in a potentially modified document.

6.  **Output:** The function returns a slice of `Anchor` structs. Each `Anchor` contains:
    *   `Position`: The starting byte position (0-indexed) of the anchor within `oldText`.
    *   `Score`: The calculated similarity score based on context matching. Higher scores indicate a better match.
    *   `Line`: The 1-indexed line number where the anchor begins.

## Use Cases

*   **Diff/Patch Application:** When applying a change (like deleting or modifying `searchText` at `originalChangeStartPos`) to a slightly different version of `oldText`, the highest-scoring anchor can indicate the correct place to apply the change.
*   **Collaborative Editing:** Identifying corresponding text segments across different users' versions of a document.
*   **Code Analysis/Refactoring:** Finding similar code snippets based on content and context.

## Diagram

See the [Anchoring Logic Diagram](docs/anchoring_logic.d2) for a visual representation of the process.

## Usage

```bash
go run main.go
``` 