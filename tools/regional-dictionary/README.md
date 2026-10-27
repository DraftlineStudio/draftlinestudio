# Regional vocabulary pipeline

Builds the style engine's regional-vocabulary dictionary from open data,
so nobody has to know by heart where a bubbler is a drinking fountain.

## Sources and licenses

- **Wiktionary**, via the [wiktextract](https://github.com/tatuylonen/wiktextract)
  raw dump hosted at kaikki.org. The data is CC BY-SA / GFDL (Wiktionary's
  own terms); the generated dictionary inherits that license and must ship
  with attribution beside it.
- **eWAVE** (https://ewave-atlas.org, CC BY 3.0) is the reference for the
  dialect-construction recognizers; it is read, not ingested here.
- DARE and the Yale Grammatical Diversity Project are reading references
  only — never ingested, for licensing.

## Workflow

1. Download `raw-wiktextract-data.jsonl.gz` from kaikki.org (≈3 GB; it is
   not kept in the repository).
2. `python3 -I extract.py <dump.jsonl.gz> --dic <en_US.dic> --roster
   publish/third-party-roster.json --out tools/regional-dictionary`
   streams the dump and writes `candidates.json` + `candidates.md`.
   Candidates are screened against the publish name guard at this step; a
   shipped data file cannot be unpublished, so collisions never get in.
3. Review `candidates.md`; add any unwanted word to `rejected.txt`, one per
   line. `(!)` marks words with common everyday senses (pop, carriage) that
   the engine treats as context-dependent.
4. The build step (with the style engine) folds `candidates.json` minus
   `rejected.txt` into the bundled dictionary, with the license README.
