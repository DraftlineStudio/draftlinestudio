package readaloud

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
)

// These structures mirror sherpa-onnx's stable C API at v1.13.7. All native
// targets Draftline supports here are 64-bit, so uintptr matches C pointers.
// Unused model-family fields must remain present: they are part of the ABI.
type nativeVitsConfig struct {
	Model, Lexicon, Tokens, DataDir      uintptr
	NoiseScale, NoiseScaleW, LengthScale float32
	_                                    uint32
	DictDir                              uintptr
}
type nativeMatchaConfig struct {
	AcousticModel, Vocoder, Lexicon, Tokens, DataDir uintptr
	NoiseScale, LengthScale                          float32
	DictDir                                          uintptr
}
type nativeKokoroConfig struct {
	Model, Voices, Tokens, DataDir uintptr
	LengthScale                    float32
	_                              uint32
	DictDir, Lexicon, Lang         uintptr
}
type nativeKittenConfig struct {
	Model, Voices, Tokens, DataDir uintptr
	LengthScale                    float32
	_                              uint32
}
type nativeZipvoiceConfig struct {
	Tokens, Encoder, Decoder, Vocoder, DataDir, Lexicon uintptr
	FeatScale, TShift, TargetRMS, GuidanceScale         float32
}
type nativePocketConfig struct {
	LMFlow, LMMain, Encoder, Decoder, TextConditioner, VocabJSON, TokenScoresJSON uintptr
	VoiceEmbeddingCacheCapacity                                                   int32
	_                                                                             uint32
}
type nativeSupertonicConfig struct {
	DurationPredictor, TextEncoder, VectorEstimator, Vocoder, TTSJSON, UnicodeIndexer, VoiceStyle uintptr
}
type nativeModelConfig struct {
	Vits       nativeVitsConfig
	NumThreads int32
	Debug      int32
	Provider   uintptr
	Matcha     nativeMatchaConfig
	Kokoro     nativeKokoroConfig
	Kitten     nativeKittenConfig
	Zipvoice   nativeZipvoiceConfig
	Pocket     nativePocketConfig
	Supertonic nativeSupertonicConfig
}
type nativeTTSConfig struct {
	Model           nativeModelConfig
	RuleFSTs        uintptr
	MaxNumSentences int32
	_               uint32
	RuleFARs        uintptr
	SilenceScale    float32
	_               uint32
}
type NativeEngine struct {
	mu           sync.Mutex
	tts          uintptr
	onnxHandle   uintptr
	sherpaHandle uintptr
	sampleRate   int

	generate     func(uintptr, string, int32, float32, uintptr, uintptr) uintptr
	destroyAudio func(uintptr)
	destroyTTS   func(uintptr)
}

type nativeCallbackState struct {
	ctx  context.Context
	emit func([]byte) error
	err  error
}

var (
	nativeCallbackOnce     sync.Once
	nativeCallbackPointer  uintptr
	nativeCallbackSequence atomic.Uintptr
	nativeCallbackStates   sync.Map
)

func sharedNativeCallback() uintptr {
	nativeCallbackOnce.Do(func() {
		// Windows' callback trampoline accepts only uintptr-sized arguments and
		// results. Those are ABI-compatible with C pointers/int32 here; avoiding
		// a float progress argument also keeps this callback portable.
		nativeCallbackPointer = purego.NewCallback(func(samples, n, key uintptr) uintptr {
			value, ok := nativeCallbackStates.Load(key)
			if !ok {
				return 0
			}
			state := value.(*nativeCallbackState)
			if n == 0 || samples == 0 {
				return 1
			}
			if err := state.ctx.Err(); err != nil {
				state.err = err
				return 0
			}
			floats := unsafe.Slice((*float32)(unsafe.Pointer(samples)), int(n))
			view := unsafe.Slice((*byte)(unsafe.Pointer(&floats[0])), int(n)*4)
			if err := state.emit(append([]byte(nil), view...)); err != nil {
				state.err = err
				return 0
			}
			return 1
		})
	})
	return nativeCallbackPointer
}

func cString(value string) ([]byte, uintptr) {
	b := append([]byte(value), 0)
	return b, uintptr(unsafe.Pointer(&b[0]))
}

func registerNativeFunctions(handle uintptr, targets ...struct {
	name string
	fn   any
}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("load sherpa symbol: %v", r)
		}
	}()
	for _, target := range targets {
		purego.RegisterLibFunc(target.fn, handle, target.name)
	}
	return nil
}

func NewNativeEngine(dir string, threads int) (*NativeEngine, error) {
	if !NativeSupported() {
		return nil, fmt.Errorf("native read aloud is unsupported on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if !Check(dir).NativeInstalled {
		return nil, fmt.Errorf("native read aloud bundle is not installed")
	}
	if threads < 1 {
		threads = 1
	}
	if threads > runtime.NumCPU() {
		threads = runtime.NumCPU()
	}

	ortName, sherpaName := nativeLibraryNames()
	runtimeDir := filepath.Join(dir, "native", "runtime")
	onnxHandle, err := openNativeLibrary(filepath.Join(runtimeDir, ortName), true)
	if err != nil {
		return nil, fmt.Errorf("load ONNX Runtime: %w", err)
	}
	sherpaHandle, err := openNativeLibrary(filepath.Join(runtimeDir, sherpaName), false)
	if err != nil {
		_ = closeNativeLibrary(onnxHandle)
		return nil, fmt.Errorf("load sherpa runtime: %w", err)
	}

	engine := &NativeEngine{onnxHandle: onnxHandle, sherpaHandle: sherpaHandle}
	var create func(*nativeTTSConfig) uintptr
	var sampleRate func(uintptr) int32
	if err := registerNativeFunctions(sherpaHandle,
		struct {
			name string
			fn   any
		}{"SherpaOnnxCreateOfflineTts", &create},
		struct {
			name string
			fn   any
		}{"SherpaOnnxDestroyOfflineTts", &engine.destroyTTS},
		struct {
			name string
			fn   any
		}{"SherpaOnnxOfflineTtsSampleRate", &sampleRate},
		struct {
			name string
			fn   any
		}{"SherpaOnnxOfflineTtsGenerateWithCallbackWithArg", &engine.generate},
		struct {
			name string
			fn   any
		}{"SherpaOnnxDestroyOfflineTtsGeneratedAudio", &engine.destroyAudio},
	); err != nil {
		engine.Close()
		return nil, err
	}

	modelPath := filepath.Join(dir, "native", "model", "model.int8.onnx")
	voicesPath := filepath.Join(dir, "native", "model", "voices.bin")
	tokensPath := filepath.Join(dir, "native", "model", "tokens.txt")
	lexiconPath := filepath.Join(dir, "native", "model", "lexicon-us-en.txt")
	dataPath := filepath.Join(dir, "native", "model", "espeak-ng-data")
	modelBytes, modelPtr := cString(modelPath)
	voicesBytes, voicesPtr := cString(voicesPath)
	tokensBytes, tokensPtr := cString(tokensPath)
	lexiconBytes, lexiconPtr := cString(lexiconPath)
	dataBytes, dataPtr := cString(dataPath)
	providerBytes, providerPtr := cString("cpu")
	config := nativeTTSConfig{}
	config.Model.NumThreads = int32(threads)
	config.Model.Provider = providerPtr
	config.Model.Kokoro = nativeKokoroConfig{
		Model: modelPtr, Voices: voicesPtr, Tokens: tokensPtr,
		DataDir: dataPtr, Lexicon: lexiconPtr, LengthScale: 1,
	}
	config.MaxNumSentences = 1
	config.SilenceScale = 0.2
	engine.tts = create(&config)
	// The native constructor consumes these strings synchronously.
	runtime.KeepAlive(modelBytes)
	runtime.KeepAlive(voicesBytes)
	runtime.KeepAlive(tokensBytes)
	runtime.KeepAlive(lexiconBytes)
	runtime.KeepAlive(dataBytes)
	runtime.KeepAlive(providerBytes)
	if engine.tts == 0 {
		engine.Close()
		return nil, fmt.Errorf("sherpa rejected the installed Kokoro model")
	}
	engine.sampleRate = int(sampleRate(engine.tts))
	if engine.sampleRate <= 0 {
		engine.Close()
		return nil, fmt.Errorf("sherpa returned invalid sample rate %d", engine.sampleRate)
	}
	return engine, nil
}

// Synthesize emits bounded PCM blocks as soon as sherpa produces them. The
// engine is intentionally serialized: ONNX sessions are kept hot but are not
// re-entered, and cancellation propagates through the native callback.
func (e *NativeEngine) Synthesize(ctx context.Context, text, voice string, speed float32, emit func([]byte) error) error {
	if text == "" {
		return fmt.Errorf("text is empty")
	}
	if speed < 0.5 || speed > 2 {
		return fmt.Errorf("speed is out of range")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tts == 0 {
		return fmt.Errorf("native read aloud engine is closed")
	}

	key := nativeCallbackSequence.Add(1)
	state := &nativeCallbackState{ctx: ctx, emit: emit}
	nativeCallbackStates.Store(key, state)
	defer nativeCallbackStates.Delete(key)
	audio := e.generate(e.tts, text, int32(nativeVoiceID(voice)), speed, sharedNativeCallback(), key)
	if audio != 0 {
		e.destroyAudio(audio)
	}
	if state.err != nil {
		return state.err
	}
	if audio == 0 {
		return fmt.Errorf("native Kokoro synthesis failed")
	}
	return nil
}

func (e *NativeEngine) SampleRate() int { return e.sampleRate }

func (e *NativeEngine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tts != 0 && e.destroyTTS != nil {
		e.destroyTTS(e.tts)
		e.tts = 0
	}
	if e.sherpaHandle != 0 {
		_ = closeNativeLibrary(e.sherpaHandle)
		e.sherpaHandle = 0
	}
	if e.onnxHandle != 0 {
		_ = closeNativeLibrary(e.onnxHandle)
		e.onnxHandle = 0
	}
}

func nativeVoiceID(voice string) int {
	ids := map[string]int{
		"af_heart": 3, "af_bella": 2, "af_nicole": 6,
		"am_michael": 16, "am_fenrir": 14, "am_puck": 18,
		"bf_emma": 21, "bm_george": 26,
	}
	if id, ok := ids[voice]; ok {
		return id
	}
	return 3
}

// Compile-time ABI checks for the 64-bit sherpa v1.13.7 C structures.
var _ [448 - unsafe.Sizeof(nativeTTSConfig{})]byte
var _ [unsafe.Sizeof(nativeTTSConfig{}) - 448]byte
