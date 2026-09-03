# Analysis Concurrency and Performance

Draftline never changes the process-wide Go scheduler during manuscript
analysis. `analysis_budget.go` sizes only the ProseV3 linguistic and evidence
worker pools, and the backend retains one whole-book analysis at a time.

## Pool budgets

- Gentle: one or two workers depending on available logical CPUs.
- Balanced: half the available logical CPUs, capped at four workers.
- Fast: one worker per available logical CPU.
- Adaptive: Balanced workers. At 750,000 source bytes and above it also uses
  a 4 MiB weighted in-flight payload gate and 512 KiB linguistic batches.

The large-manuscript threshold exists to bound transient Prose document memory
multiplied by concurrent jobs. It is intentionally not a scheduler or thread-
count clamp. A source chapter larger than the payload gate stays intact so
sentence and entity offsets remain stable, but it is charged as the entire
gate and runs alone.

## Windows 12-thread validation — 2026-09-03

the benchmark manuscript was re-opened fresh and its derived analysis cleared before every
run. Its manuscript contains approximately 70,700 words and 469,876 source
bytes, so it exercises the normal tier rather than the large-manuscript memory
gate. CPU percentages are normalized to the machine's 12 logical threads.

Three direct whole-pipeline runs produced these medians:

| Configuration | Analysis wall time | Peak process CPU | Scheduler probe delay |
| --- | ---: | ---: | ---: |
| Legacy behavior simulated with process-wide scheduler = 4, pool = 4 | 5.374 s | 34.0% | 19.0 ms |
| Balanced pool = 4, scheduler left at 12 | 5.318 s | 39.7% | 1.4 ms |
| Fast pool = 12, scheduler left at 12 | 4.125 s | 89.5% | 6.9 ms |

The three Fast runs peaked at 88.4%, 89.5%, and 98.1%. None stalled the
scheduler; the worst 200 ms sampling-window wake delay was 9.6 ms.

An end-to-end Wails/WebView run exercised the actual bindings while Read Aloud
played chapter prose. During Balanced analysis, 211 probes completed:

| Concurrent operation | Mean latency | Worst observed latency |
| --- | ---: | ---: |
| `GetAppVersion` Wails binding | 3.9 ms | 64.1 ms |
| Read Aloud model asset request | 6.7 ms | 30.4 ms |
| `ReadAloudStatus` filesystem request | 4.4 ms | 69.0 ms |

The end-to-end analysis request took 15.23 s, including Wails serialization of
the complete analyzed book. Backend CPU peaked at 35.8%. Read Aloud remained
live throughout. Its second-sentence AudioContext handoff gap was 7.20–7.24 s,
compared with 4.94 s in a no-analysis control. This machine's current Kokoro
WASM configuration was already slower than real time in the control (RTF
1.62); concurrent Balanced analysis increased sentence-two generation from
11.21 s to 13.47 s. The pool isolation prevents scheduler starvation, but it
cannot make finite CPU contention disappear, so this result must not be
reported as zero TTS impact.

The full-app Fast stress run reached 96.4% combined Draftline/backend plus
isolated browser/TTS CPU (88.3% backend peak). The UI probe completed 164
cycles: binding latency averaged 7.1 ms and peaked at 109.7 ms; asset requests
averaged 7.9 ms and peaked at 139.9 ms; filesystem requests averaged 9.9 ms
and peaked at 203.5 ms. The app remained responsive and did not reproduce the
former 100%-CPU lockup, though Fast is intentionally aggressive.
