// Package readaloud manages the optional, fully local Read Aloud voice model:
// a pinned download of the Kokoro-82M ONNX model (Apache 2.0), its voice
// embeddings, and the onnxruntime-web runtime files, installed under the user
// cache directory and served to the webview by a read-only asset handler.
// After the one-time download, synthesis runs entirely offline; no text or
// audio ever leaves the machine.
package readaloud

// Artifact is one pinned file of the Read Aloud bundle. Name is the relative
// install path under the model directory; URL points at an immutable revision
// (a Hugging Face commit or an exact npm version on jsdelivr — never a mutable
// branch, per docs/architecture/PLUGIN-SYSTEM.md). Every file is verified
// against SHA256 and Bytes before it is moved into place.
//
// Group partitions the bundle: "core" is the mandatory CPU (WASM q8) set;
// "gpu" is the optional fp32 model for the WebGPU fast path, downloaded only
// when the user opts in (q8 on WebGPU produced corrupted audio; fp32 is the
// configuration that renders cleanly there).
type Artifact struct {
	Name   string
	URL    string
	SHA256 string
	Bytes  int64
	Group  string
}

const (
	GroupCore = "core"
	GroupGPU  = "gpu"
)

// kokoroRevision is the pinned commit of onnx-community/Kokoro-82M-v1.0-ONNX.
const kokoroRevision = "1939ad2a8e416c0acfeecc08a694d14ef25f2231"

// ortVersion matches the onnxruntime-web version resolved by the frontend's
// @huggingface/transformers dependency (see frontend/package-lock.json); the
// .mjs/.wasm pair must come from the same build the bundled runtime expects.
const ortVersion = "1.22.0-dev.20250409-89f8206ba4"

const hfBase = "https://huggingface.co/onnx-community/Kokoro-82M-v1.0-ONNX/resolve/" + kokoroRevision + "/"
const ortBase = "https://cdn.jsdelivr.net/npm/onnxruntime-web@" + ortVersion + "/dist/"

// hfDir is where the Hugging Face repo files land, mirroring the repo layout so
// transformers.js localModelPath resolution finds them unchanged.
const hfDir = "hf/onnx-community/Kokoro-82M-v1.0-ONNX/"

func hf(rel, sha string, bytes int64) Artifact {
	return Artifact{Name: hfDir + rel, URL: hfBase + rel, SHA256: sha, Bytes: bytes, Group: GroupCore}
}

func gpuHF(rel, sha string, bytes int64) Artifact {
	a := hf(rel, sha, bytes)
	a.Group = GroupGPU
	return a
}

func ort(file, sha string, bytes int64) Artifact {
	return Artifact{Name: "ort/" + file, URL: ortBase + file, SHA256: sha, Bytes: bytes, Group: GroupCore}
}

// Manifest returns the complete pinned artifact list. Checksums were computed
// from the pinned revisions at integration time (voice/model hashes match the
// repo's LFS records).
func Manifest() []Artifact {
	return []Artifact{
		hf("config.json", "df34b4f930b23447cd4dc410fabfb42eb3f24e803e6c3f97d618fb359380a36f", 44),
		hf("tokenizer.json", "77a02c8e164413299b4b4c403b14f8e0e1c1b727db4d46a09d6327b861060a34", 3497),
		hf("tokenizer_config.json", "be1cb066d6ef6b074b3f15e6a6dd21ac88ff3cdaedf325f0aaed686c70f75d20", 113),
		hf("onnx/model_quantized.onnx", "fbae9257e1e05ffc727e951ef9b9c98418e6d79f1c9b6b13bd59f5c9028a1478", 92361116),
		hf("voices/af_heart.bin", "d583ccff3cdca2f7fae535cb998ac07e9fcb90f09737b9a41fa2734ec44a8f0b", 522240),
		hf("voices/af_bella.bin", "f69d836209b78eb8c66e75e3cda491e26ea838a3674257e9d4e5703cbaf55c8b", 522240),
		hf("voices/af_nicole.bin", "cd2191ab31b914ed7b318416b0e4440fdf392ddad9106a060819aa600a64f59a", 522240),
		hf("voices/am_michael.bin", "1d1f21dd8da39c30705cd4c75d039d265e9bc4a2a93ed09bc9e1b1225eb95ba1", 522240),
		hf("voices/am_fenrir.bin", "c27989f741f7ee34d273a39d8a595cc0837d35f5ced9a29b7cc162614616df43", 522240),
		hf("voices/am_puck.bin", "fcf73c989033e9233e0b98713eca600c8c74dcc1614b37009d5450ff4a2274a0", 522240),
		hf("voices/bf_emma.bin", "669fe0647f9dd04fcab92f1439a40eeb4c8b4ab1f82e4996fe3d918ce4a63b73", 522240),
		hf("voices/bm_george.bin", "c4b235a4c1f2cd3b939fed08b899ce9385638b763f7b73a59616c4fc9bd6c9bc", 522240),
		ort("ort-wasm-simd-threaded.mjs", "43c25054b6b9ac000f786c65545ff83a45f871e0e310e8c2f4d48a363bb66db4", 20856),
		ort("ort-wasm-simd-threaded.wasm", "f061472c6e77d6d50d079aacdc0ff9b63fee287ddd2cbf46cf62438d3891de2b", 11133407),
		ort("ort-wasm-simd-threaded.jsep.mjs", "08fb86ec433c78bfb032c5d84a68b8e8e5a8d81268fa39e24314179a5767a5b9", 44484),
		ort("ort-wasm-simd-threaded.jsep.wasm", "c46655e8a94afc45338d4cb2b840475f88e5012d524509916e505079c00bfa39", 21596019),
		// Optional WebGPU fast path: full-precision model (~311 MB).
		gpuHF("onnx/model.onnx", "8fbea51ea711f2af382e88c833d9e288c6dc82ce5e98421ea61c058ce21a34cb", 325532232),
	}
}

// GroupManifest returns the artifacts of one group.
func GroupManifest(group string) []Artifact {
	var result []Artifact
	for _, a := range Manifest() {
		if a.Group == group {
			result = append(result, a)
		}
	}
	return result
}

func groupBytes(group string) int64 {
	var total int64
	for _, a := range GroupManifest(group) {
		total += a.Bytes
	}
	return total
}

// TotalBytes is the size of the mandatory core bundle (the number shown in
// download prompts); the GPU group is priced separately.
func TotalBytes() int64 {
	return groupBytes(GroupCore)
}
