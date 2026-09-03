// Package readaloud manages Draftline's checksum-pinned, fully local native
// Kokoro voice bundle. No browser inference model or runtime is installed.
package readaloud

import "runtime"

type Artifact struct {
	Name   string
	URL    string
	SHA256 string
	Bytes  int64
	Group  string
}

const GroupNative = "native"

const (
	sherpaVersion      = "v1.13.7"
	nativeRevision     = "7e9b67b79bfdcbd2b4bc144370345fcceac3cb0c"
	nativeInt8Revision = "5d6cbe65546edb3ebae8bde976c8ad3438b3f34b"
	espeakBundleURL    = "https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/espeak-ng-data.tar.bz2"
)

func nativeModel(file, sha string, bytes int64) Artifact {
	return Artifact{
		Name:   "native/model/" + file,
		URL:    "https://huggingface.co/csukuangfj/kokoro-multi-lang-v1_0/resolve/" + nativeRevision + "/" + file,
		SHA256: sha, Bytes: bytes, Group: GroupNative,
	}
}

func nativeInt8Model() Artifact {
	return Artifact{
		Name:   "native/model/model.int8.onnx",
		URL:    "https://huggingface.co/csukuangfj/kokoro-int8-multi-lang-v1_0/resolve/" + nativeInt8Revision + "/model.int8.onnx",
		SHA256: "77ef4f0513401d508ed7831f8504c7042df58bc75e004ec9666894590f999b1d",
		Bytes:  114298054, Group: GroupNative,
	}
}

func nativeRuntime(repo, triple, file, sha string, bytes int64) Artifact {
	return Artifact{
		Name:   "native/runtime/" + file,
		URL:    "https://raw.githubusercontent.com/k2-fsa/" + repo + "/" + sherpaVersion + "/lib/" + triple + "/" + file,
		SHA256: sha, Bytes: bytes, Group: GroupNative,
	}
}

// Only libraries runnable by this process are downloaded. Unsupported targets
// fail explicitly rather than acquiring a foreign or unusable runtime.
func nativeRuntimeManifest() []Artifact {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/amd64":
		return []Artifact{
			nativeRuntime("sherpa-onnx-go-windows", "x86_64-pc-windows-gnu", "onnxruntime.dll", "b9f6713c3602a4742680a7e6a77e3f9ac4a676ad9447bce609e53efcda795d7e", 17378304),
			nativeRuntime("sherpa-onnx-go-windows", "x86_64-pc-windows-gnu", "sherpa-onnx-c-api.dll", "ca7912b726d58c9fc324169f57b8b7b12171b981bd1570a108dd06b9a8f17a15", 4593664),
		}
	case "linux/amd64":
		return []Artifact{
			nativeRuntime("sherpa-onnx-go-linux", "x86_64-unknown-linux-gnu", "libonnxruntime.so", "c85f471e1bd5059a4556038f7f5288fa41141647613688452ae7de4879150903", 26407985),
			nativeRuntime("sherpa-onnx-go-linux", "x86_64-unknown-linux-gnu", "libsherpa-onnx-c-api.so", "5857fcb0d39041902eef1b6fb2165cdd48b6c6f7de2ea951c992a4628a1f1b04", 5102560),
		}
	case "linux/arm64":
		return []Artifact{
			nativeRuntime("sherpa-onnx-go-linux", "aarch64-unknown-linux-gnu", "libonnxruntime.so", "d860f5968f5a1ed63533e9ed198aa747ca9fe289028129877f428556089f6874", 33951528),
			nativeRuntime("sherpa-onnx-go-linux", "aarch64-unknown-linux-gnu", "libsherpa-onnx-c-api.so", "5c8d2884e451750d8b5e680d38efd044dea596ac1e2ffcdbc17886c4e34e8228", 4610856),
		}
	case "darwin/amd64":
		return []Artifact{
			nativeRuntime("sherpa-onnx-go-macos", "x86_64-apple-darwin", "libonnxruntime.dylib", "8dc3d87827183d659f65fc526bd9ea06ad7e9bd906b16bdae26338d3ef26863e", 32172656),
			nativeRuntime("sherpa-onnx-go-macos", "x86_64-apple-darwin", "libsherpa-onnx-c-api.dylib", "a2341ad1843f7267e83742291b5b9005ba72559e2de53f81e757cfd619794468", 4470192),
		}
	case "darwin/arm64":
		return []Artifact{
			nativeRuntime("sherpa-onnx-go-macos", "aarch64-apple-darwin", "libonnxruntime.dylib", "f1217a893bd4e619022264dbc4bad3490866552b48926ee4c09243e9200b4c53", 28942064),
			nativeRuntime("sherpa-onnx-go-macos", "aarch64-apple-darwin", "libsherpa-onnx-c-api.dylib", "9e26d7ec53650b622adf0e9d4b16863cbc54016e619d89b70213ecec1dc1af91", 4159560),
		}
	default:
		return nil
	}
}

func Manifest() []Artifact {
	manifest := []Artifact{
		nativeModel("voices.bin", "8a77c0d397026208d22211f37670b5b3b11e03f190756b25a1d24041fced82a9", 27678720),
		nativeModel("tokens.txt", "6ebb6bb288f20f3ae8d004d3c2ca27697da27c037d75e81a60e2a6a663f95425", 687),
		nativeModel("lexicon-us-en.txt", "7daaab53a181be9885b853a8582bf1838186317e5dadacbcef9c426d6fa0da14", 5956885),
		{Name: "native/model/espeak-ng-data.tar.bz2", URL: espeakBundleURL, SHA256: "4135ccf82e1f40613491c0874d4945ae9e9c7840933d8e25a6f9e003d9ebf533", Bytes: 7252012, Group: GroupNative},
		nativeInt8Model(),
	}
	return append(manifest, nativeRuntimeManifest()...)
}

func GroupManifest(group string) []Artifact {
	if group != GroupNative {
		return nil
	}
	return Manifest()
}

func NativeBytes() int64 {
	var total int64
	for _, artifact := range Manifest() {
		total += artifact.Bytes
	}
	return total
}

// TotalBytes is retained as the general bundle-size API; native is now the
// complete and only Read Aloud bundle.
func TotalBytes() int64 { return NativeBytes() }

func NativeSupported() bool { return len(nativeRuntimeManifest()) != 0 }
