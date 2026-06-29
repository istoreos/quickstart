package service

// Guide SoftSource currently shares the same file-based seam implementation
// between read and write paths. This file intentionally hosts the write-side
// entrypoint so later tasks can switch wrappers without reworking imports.
