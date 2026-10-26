//go:build nervos_echo && !dot && !spot && !panel

package update

// NervOS Echo uses the existing signed-manifest and A/B rootfs updater.
// Runtime settings cannot redirect this root-trusted release origin.
const releases = "https://github.com/hblake-xcantech/techo5/releases"
