# NervOS Echo — first increment

This runtime is the Echo Linux platform of the existing Nerve project. It is a
history-preserving TECHO5 fork, not a separate household identity or policy system.
The Nerve superproject owns device enrollment, context/action policy, Orbit and
shared design language. This fork owns Echo hardware, rendering and recoverable updates.
TECHO5's MIT license and contributor attribution remain intact.

Base: `56452227c69a16e12d654a8c62ed40ca7440810f`.
Build the Echo variant with `-tags nervos_echo`; ordinary upstream builds keep their
original release origin and trust key. The fork variant has a distinct compiled
release origin and Ed25519 public key. Private signing material is never in Git.
A fork release cannot silently replace itself with upstream's next release.

## Update boundary

| Change | Existing mechanism | Device reboot |
|---|---|---|
| Supported runtime settings | Feature setters and persistent config store | No |
| Validated shared-room text after feature installation | Fixed operator-owned projection file, atomic revision/expiry checks | No; trusted transport/enrollment pending |
| Compiled renderer/daemon on this Linux slot image | Signed rootfs into inactive A/B slot | Yes |
| Kernel/boot image | Separate board-specific recovery process | Yes |

Do not claim arbitrary source/UI changes are rebootless. The native [shared-room presentation](nervos-presentation/README.md) validates an atomic
revision before swapping it, retains the last good revision until its original expiry,
preserves manually changed controls, and interprets no executable content or action URLs.
Its Linux previews and tests pass; it has not yet been installed on the Echo.
Menus remain available for recovery and explicit control; normal use should be an
AI interaction, with useful context and restrained attention cues.

## Existing recovery retained

The signed-manifest verifier, strict update TLS client, hash/size checks, download
resume, install lock, vendor preservation and slotctl are unchanged. An update
unpacks into the inactive slot, marks it trial for three boots and reboots. The
boot path falls back to a known good slot when trials run out. A stable daemon
commits the new slot after five minutes. There is no need to repeat the Amazon
unlock for normal rootfs releases. This is not protection from every bootloader,
flash-storage or power failure.

The first transition from upstream TECHO5 to this fork must use an authenticated
operator path, verify the fork signature and hash, retain the current good slot,
and check quiet ownership before its necessary reboot. Do not replace upstream's
trust key in a running device as a shortcut. No custom release is installed yet.

Before a live deployment: build and verify a signed fork artifact, exercise an
inactive-slot trial and rollback in a non-device fixture, verify the live prior
slot is good and backups exist, preserve device config, and coordinate the brief
reboot. A host test is not proof of physical recovery.

## Upstream maintenance

Keep `upstream` pointed at HuskerMinion/techo5 and `origin` at the fork. Merge an
explicit upstream revision into the NervOS branch, run update/config/rollback
checks, review the small patch set, and advance the superproject gitlink. Do not
sync running devices automatically from upstream HEAD.

## Reproducible candidate assembly

`tools/nervos-rootfs.py` downloads and verifies the pinned upstream v1.6.8 signed
manifest, checks its rootfs hash, and replaces only the compiled ARM daemon and
release label. It preserves archive ownership, modes, links and recovery files;
it never extracts an archive or touches hardware. It refuses a wrong base hash,
wrong architecture, unsafe path, missing replacement member or existing output.
Use a version newer than the upstream base, such as `v1.6.9-nervos.1`, to preserve
the updater's downgrade guard. Build the daemon with `-tags nervos_echo`, then use
upstream's `cmd/mkmanifest` and an operator-held signing seed through stdin to sign
the candidate. Verify the signature and every artifact hash before any upload.
This builder does not publish or install anything.

Run `python3 tools/test_nervos_slots.py` and `python3 tools/test_nervos_rootfs.py`,
plus `go test -tags nervos_echo ./internal/update/... ./internal/config` from `echod`.
The slot tests run the real slotctl on disposable directory stores, not a device.
