---
date: 2026-10-13
kind: chore
---

# Move the doc caps into a limits.json once a third enforcer exists

The doc caps are read by the hooks and the rule, and by the pusher. A `limits.json` read by all of
them would keep the numbers in one place. Do this only when a third enforcement point appears; the
readers do not justify the file.
