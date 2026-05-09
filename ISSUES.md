# Known Issues

This document tracks known issues identified during code review.

- **Tempo comment source**: Currently the Tempo worklog comment is built by aggregating unique time-entry comments. When time entries have no comments, the worklog is sent with an empty description. Consider whether the task description should be used as a fallback comment, or whether a per-task `X-WORKLOG-TEMPO-COMMENT` override property should be added.

---

*Last reviewed: 2026-05-09*
