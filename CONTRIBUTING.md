# Contributing to Draftline

Thanks for your interest in Draftline. 1.0 is released and the app is stable;
development is still quick, so a feature area can change shape between builds.

## Getting set up

Follow [Building from source](README.md#building-from-source) in the README.
In short: Go 1.25+, Node 22+, and the Wails v2 CLI; then `wails dev` from
`wails/` for a live-reload development build.

## Before you open a PR

- `cd wails && go test ./...` — Go suite, including the file-size guardrail
- `cd wails/frontend && npm test` — frontend (Vitest) suite
- `cd wails/frontend && npm run build` — TypeScript check + production build

All three must pass. New behavior should come with tests beside the code it
exercises (`*.test.ts` next to the module, `*_test.go` in the package).

## House rules worth knowing

- **One stylesheet.** `wails/frontend/src/styles/global.css` is the only CSS
  file, on purpose. Never add a co-located `.css` file or a CSS import to a
  component — new feature styles go in global.css as a bannered section. The
  file's header explains why.
- **File-size guardrail.** `go test` fails if a source file crosses the
  800-line threshold (or its recorded allowance). Prefer splitting;
  raising an allowance needs a written reason in the PR.
- **Versioning.** Every user-visible change bumps the build number and adds a
  CHANGELOG entry. `docs/VERSION.md` lists every location the version string
  lives — update all of them together.
- **Privacy is a feature.** Manuscript text must never leave the machine
  except through the user's explicitly configured AI provider. Analysis,
  indexing, spelling/grammar, and Read Aloud are local and must stay local.
- **No manuscripts in the repo.** `*.draftline` files and reference assets
  are gitignored; keep test fixtures synthetic.

## Reporting bugs

Help → Report an Issue in the app opens a new issue with your version already
filled in. Otherwise open one by hand with your OS, the app version (Settings
footer), and steps to reproduce. For crashes, the terminal output from
`wails dev` is gold.
