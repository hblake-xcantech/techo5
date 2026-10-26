//go:build nervos_echo && !dot && !spot && !panel

package update

// Only the public key is distributed. The release seed remains with the operator.
// A fork build must never accept an upstream-signed replacement accidentally.
var releaseKey = "YaRDVWvH1cWaWeqguO1HY2wbGWh9TL/Pxd1kOfPVRLA="
