# Known Issues

This document tracks known issues identified during code review.

### `time edit` cannot set event description
**Priority:** Low
**Impact:** `EventUpdate` has a `Description` field, but the CLI only exposes `-note` which maps to `Summary`.

- Add a `-description` flag to `time edit` (and possibly `time add`) to populate `Event.Description`.

---

*Last reviewed: 2026-05-06*
