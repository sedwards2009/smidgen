package clipboard

import (
	"os"
	"runtime"

	"github.com/atotto/clipboard"
)

var internalClipboard string
var useInternalClipboard bool

func ClipboardReadAll() (string, error) {
	if useInternalClipboard {
		return internalClipboard, nil
	}
	return clipboard.ReadAll()
}

func ClipboardWriteAll(s string) error {
	if useInternalClipboard {
		internalClipboard = s
		return nil
	}
	return clipboard.WriteAll(s)
}

func isUnix() bool {
	return runtime.GOOS == "linux" || runtime.GOOS == "openbsd" || runtime.GOOS == "netbsd"
}

func isX11Present() bool {
	return os.Getenv("DISPLAY") != ""
}

func isWaylandPresent() bool {
	return os.Getenv("WAYLAND_DISPLAY") != ""
}

func init() {
	useInternalClipboard = clipboard.Unsupported

	if !clipboard.Unsupported && isUnix() && !(isWaylandPresent() || isX11Present()) {
		useInternalClipboard = true
	}
}
