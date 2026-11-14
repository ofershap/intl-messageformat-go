# Security Policy

Report vulnerabilities privately via GitHub's "Report a vulnerability" flow
on this repository (Security tab). Please do not open public issues for
security reports.

This package parses and renders message templates with caller-supplied
values. It performs no network access, file I/O, or reflection over caller
types beyond documented value kinds. Untrusted message strings are an input
worth fuzzing; the corpus and fuzz targets live in the test suite.

## Input bounds

- Message nesting is limited to 100 levels; deeper nesting returns a typed
  parse error (covered by a dedicated test and the fuzz target).
- The parser is linear-time in message length with no regex backtracking.
- Output length: each placeholder on the rendered path expands to exactly one
  formatted value, and plural/select bodies render at most one branch per
  level, so output length is O(message_length x longest formatted value).
  Repeated placeholders repeat the caller's value, which is caller-controlled;
  no internal amplification beyond that factor exists.
- FuzzParse covers arbitrary byte input; CI runs it for 30s per change and
  the corpus grows with any found input.
