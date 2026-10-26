//go:build !nervos_echo

package update

// releaseKey is the public half of the key releases are signed with (tools/release.ps1 signs
// manifest.json into manifest.json.sig). An update installs as root, so a manifest is believed only
// when this key signed it: HTTPS alone would let anything that can present a certificate the device
// accepts, or a device told to skip certificate checks, hand it a root filesystem.
var releaseKey = "KVUuQUbhyKwPBbIneqFEvXYSI+3Hkfu/heCTy5YNVMk="
