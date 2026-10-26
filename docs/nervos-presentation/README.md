# NervOS shared-room presentation, first native slice

Owner: NerveOS Echo adapter. Authorization: Blake's October 26 request for an AI/context-first Echo inside NerveOS, with routine UI/content updates without reboot where feasible. This patch changes only the fork's Show display and an isolated context component. It does not deploy to a device.

## What runs

With `nervos_echo && !dot && !spot && !panel`, importing the display registers the `nervoscontext` component through the upstream component lifecycle. Its one-second loop reads a fixed local operator directory. The idle renderer calls `nervosContext` from `render.go`; the existing display cadence shows accepted revisions and expiry on the next idle frame. No reboot, process restart, HTTP listener, action endpoint or framebuffer reopening is required for those data changes after installation. Installing the compiled feature still uses the normal signed A/B update and reboot.

Two independent opt-ins are required:

1. The operator commissions a root-owned private `/data/nervos/presentation` directory, containing root-owned private regular files `binding.json`, `revision.json`, and `projection.json`. The feature does not create the directory or binding. Symlinks and group/other-accessible files are rejected. Read descriptors are checked against the pre-open inode and owner. The root operator is trusted; this is not a sandbox against a malicious root process.
2. The person selects **NervOS context** in the existing **Clock style** setting (also exposed by the existing HA select). Projection content cannot set this choice. Another clock style, any skin, photos, night dimming, voice, foreground pages, timers, calendar items, notifications, media or the drawer yield to upstream behavior. Brightness, microphone mute, volume and audio routes are never written.

When context is missing, expired, too large for the selected human font, or disabled, the existing Classic clock is the fallback for the NervOS choice. All other clock selections stay as selected. There are no new tap zones; upstream tap-to-talk and settings gestures remain in charge. This slice deliberately does not redesign the active voice screen.

## Local projection contract

Synthetic binding:

```json
{"deviceId":"echo-example","surfaceId":"shared-room","source":"operator-example"}
```

Initial watermark, created deliberately by the commissioning operator, not inferred from a projection:

```json
{"binding":{"deviceId":"echo-example","surfaceId":"shared-room","source":"operator-example"},"revision":0}
```

Projection (timestamps are illustrative; these will be stale at a later time):

```json
{"schemaVersion":1,"deviceId":"echo-example","surfaceId":"shared-room","source":"operator-example","sourceRevision":1,"observedAt":1800000000,"expiresAt":1800000060,"visibility":"shared","summary":"The workbench is ready for the next step.","why":"Shared context from the room operator."}
```

Replace `projection.json` with an atomic rename from a same-directory mode-0600 temporary file. Source/device/surface must match the operator binding. Only `shared` visibility is accepted; there is no private/adult/child identity override. A source label or speaker identity is not authorization. The trusted upstream producer remains responsible for classifying actual content; a shared label cannot prove that prose contains no private information.

Revisions are positive integers and strictly increase for this bound source. Observation cannot be in the future; expiry must be in the future and at most 15 minutes after observation. Summary is 1–160 Unicode code points; explanation is 1–90. Unknown/duplicate top-level keys, multiple JSON documents, invalid UTF-8, control/bidirectional-format characters and input over 4096 bytes are rejected. Content is plain text, never HTML, script, image, action URL or a command. A URL in text has no action behavior.

The full candidate is validated before swapping. The durable revision watermark is atomically renamed and synced **before** publishing a revision. Persistence failure retains the prior state. Invalid/missing projections retain the last good value only until its original expiry; expiry is rechecked on every snapshot, and context whose observation becomes future after a backward clock jump is hidden. Replay, reconnect and process restart never replay the last consumed revision. After restart, the operator must publish a newer revision. Missing/corrupt watermark or missing/invalid binding disables presentation. Deleting a projection is not a revoke: remove the binding or choose another clock style for immediate disable (within the normal poll/frame interval).

The binding and initial watermark are commissioning artifacts. Changing a binding requires a matching deliberately initialized watermark; do not reset a watermark merely to reuse old revisions. This is an operator-owned stream, not multiple competing writers.

## Design and proof

960×480 native raster. Slate `#293b46`, Cloud `#edf0ed` and Signal `#91a99a` come from canonical NerveTheme; hierarchy follows NervOS Home's type-first context paragraph. A fine rule and generous stationary spacing replace cards and app tiles. Four summary lines and two explanation lines fit within the reserved body; oversized selected fonts fall back rather than truncate. The NervOS palette is an explicit addition to the existing theme selector, never a default mutation. The context renderer uses the current human-selected palette and font; custom colors and light mode still apply. Presence timing and audio assets are untouched; no motion or sound is introduced, so reduced-motion and unknown speech coordination receive the same stable result.

Run from `echod/`:

```sh
go test -race -tags nervos_echo ./internal/feature/nervoscontext
# On a Linux host, with a scratch output directory:
SHOW_PREVIEW=/tmp/nervos-preview go test -tags nervos_echo ./internal/feature/display -run '^TestNervos' -count=1
```

Tests use synthetic text and local scratch files. The live-file lifecycle test uses an injected local reader to run unprivileged; production ownership/symlink rejection is tested separately. Preview images come from the actual native renderer, not a browser approximation. See `job.json` and `verification.json` for version-bound results and limits.

No live Orbit transport, signature verifier, enrollment proof, action broker or child activation is connected. The next integration owner must implement a trusted operator projection writer that validates upstream audience/grants and writes only this device's shared-room projection. The device reader trusts OS root ownership; it does not claim payload signatures. Do not expose its files to voice-model tools. Commission that writer and opt-in separately, then verify actual screen rendering, manual-control preservation and timed expiry on target hardware.

Rollback: select any existing clock style or remove the binding; remove this isolated feature patch for source rollback. Upstream builds have a no-op hook and no context component. Existing TECHO5 configuration and signed A/B updater remain authoritative.
