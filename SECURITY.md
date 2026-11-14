# Security Policy

Report vulnerabilities privately via GitHub's "Report a vulnerability" flow
on this repository (Security tab). Please do not open public issues for
security reports.

This package parses and renders message templates with caller-supplied
values. It performs no network access, file I/O, or reflection over caller
types beyond documented value kinds. Untrusted message strings are an input
worth fuzzing; the corpus and fuzz targets live in the test suite.
