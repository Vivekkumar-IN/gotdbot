package tdjson

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	tdCreateClientId func() int32
	tdSend           func(int32, *byte)
	tdReceive        func(float64) uintptr
	tdExecute        func(*byte) uintptr

	libLoaded bool
)

// Init initializes the TDLib JSON interface by loading the library.
func Init(libPath, version string) error {
	if libLoaded {
		return nil
	}

	if libPath != "" {
		if !filepath.IsAbs(libPath) {
			abs, err := filepath.Abs(libPath)
			if err != nil {
				return fmt.Errorf("failed to resolve tdjson binary path: %w", err)
			}
			libPath = abs
		}
		bv, err := getTDLibVersion(libPath)
		if err != nil {
			return err
		}
		if bv != version {
			return fmt.Errorf(
				"tdlib version mismatch: expected %s but binary is %s; please provide the correct tdjson binary",
				version, bv,
			)
		}
	} else {
		libPath = getLibPath(version)
		if libPath == "" {
			return fmt.Errorf("tdjson library not found for version %s; provide the correct binary in the client options", version)
		}
	}

	lib, err := purego.Dlopen(libPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return fmt.Errorf("failed to load tdjson library from %s: %w", libPath, err)
	}

	purego.RegisterLibFunc(&tdCreateClientId, lib, "td_create_client_id")
	purego.RegisterLibFunc(&tdSend, lib, "td_send")
	purego.RegisterLibFunc(&tdReceive, lib, "td_receive")
	purego.RegisterLibFunc(&tdExecute, lib, "td_execute")

	libLoaded = true

	// disable internal TDLib logging
	Execute(`{"@type": "setLogStream", "log_stream": {"@type": "logStreamEmpty"}}`)

	return nil
}
func getDefaultLibName() string {
	switch runtime.GOOS {
	case "windows":
		return "tdjson.dll"
	case "darwin":
		return "libtdjson.dylib"
	default:
		return "libtdjson.so"
	}
}

// getLibPath scans the working directory for a tdjson lib matching version.
// It checks files with the pattern "<libName>.<anything>" first,
// then falls back to the bare libName if nothing matched.
func getLibPath(version string) string {
	libName := getDefaultLibName()

	entries, err := os.ReadDir(".")
	if err != nil {
		return ""
	}

	prefix := libName + "."

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		abs, err := filepath.Abs(name)
		if err != nil {
			continue
		}

		bv, err := getTDLibVersion(abs)
		if err != nil {
			continue
		}

		if bv == version {
			return abs
		}
	}

	// fallback: try bare libName (e.g. libtdjson.so with no version suffix)
	abs, err := filepath.Abs(libName)
	if err != nil {
		return ""
	}

	bv, err := getTDLibVersion(abs)
	if err != nil || bv != version {
		return ""
	}

	return abs
}

// CreateClientID returns an opaque identifier of a new TDLib instance.
func CreateClientID() int {
	return int(tdCreateClientId())
}

// Send sends a request to the TDLib client. May be called from any thread.
func Send(clientID int, request string) {
	reqBytes := append([]byte(request), 0)
	tdSend(int32(clientID), &reqBytes[0])
}

// Receive receives incoming updates and request responses.
// Returns a JSON-serialized update or an empty string if the timeout expires.
func Receive(timeout float64) string {
	ptr := tdReceive(timeout)
	if ptr == 0 {
		return ""
	}
	return goString(ptr)
}

// Execute synchronously executes a TDLib request.
// Returns a JSON-serialized response.
func Execute(request string) string {
	reqBytes := append([]byte(request), 0)
	ptr := tdExecute(&reqBytes[0])
	if ptr == 0 {
		return ""
	}
	return goString(ptr)
}

func goString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}

	var length int
	for {
		b := *(*byte)(unsafe.Pointer(ptr + uintptr(length)))
		if b == 0 {
			break
		}
		length++
	}
	if length == 0 {
		return ""
	}

	return string(unsafe.Slice((*byte)(unsafe.Pointer(ptr)), length))
}

func getTDLibVersion(libPath string) (string, error) {
	lib, err := purego.Dlopen(libPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return "", fmt.Errorf("failed to load tdjson library from %s: %w", libPath, err)
	}

	purego.RegisterLibFunc(&tdExecute, lib, "td_execute")

	resp := Execute(`{"@type":"getOption","name":"version"}`)

	var result struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return "", fmt.Errorf("failed to parse version response: %w", err)
	}

	return "v" + result.Value, nil
}
