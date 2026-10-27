#!/usr/bin/env python3
"""Build regional-vocabulary candidates from the kaikki.org Wiktionary extract.

Streams the raw wiktextract dump (all languages, gzipped JSONL), keeps
English senses that carry a regional label plus a usable broader
alternative, and emits a candidate list for editorial review. Nothing here
ships directly: the reviewed survivors become the style engine's bundled
dictionary, and every candidate is screened against the publish name guard
first, because a bad entry in a shipped data file cannot be unpublished.

Input data is CC BY-SA / GFDL (Wiktionary, via wiktextract/kaikki.org); the
generated dictionary inherits that license and ships with attribution.

Usage:
    python3 -I extract.py RAW.jsonl.gz \
        --dic wails/frontend/public/dictionaries/en_US.dic \
        --roster publish/third-party-roster.json \
        --out tools/regional-dictionary
"""
import argparse
import gzip
import json
import pathlib
import re
import sys
from collections import defaultdict

# Regional labels worth an entry, mapped from wiktextract tags and raw label
# text. Bare "US" is deliberately absent: for a US-market manuscript it
# flags nothing, and it floods the list.
REGIONS = {
    "US-NewEngland": r"New England",
    "US-South": r"Southern US|Southern United States|US South|Southeastern US",
    "US-Midwest": r"Midwest|Upper Midwestern|Northern US",
    "US-Appalachia": r"Appalachia",
    "US-Pennsylvania": r"Pennsylvania|Philadelphia|Pittsburgh",
    "US-Wisconsin": r"Wisconsin",
    "US-Texas": r"Texas",
    "US-California": r"California",
    "US-Louisiana": r"Louisiana|New Orleans",
    "US-NewYork": r"New York",
    "US-Boston": r"Boston",
    "US-Hawaii": r"Hawaii",
    "US-West": r"(?<!Mid)western US",
    "UK": r"\bUK\b|British|(?<!New )England|Britain",
    "Scotland": r"Scottish|Scotland",
    "Ireland": r"Irish|Ireland",
    "Wales": r"Welsh|Wales",
    "Australia": r"Australian?\b",
    "NewZealand": r"New Zealand",
    "Canada": r"Canad(?:a|ian)\b",
    "India": r"Indian English|South Asia",
    "SouthAfrica": r"South Africa",
}
REGION_RES = {code: re.compile(pattern, re.IGNORECASE) for code, pattern in REGIONS.items()}

# Senses wearing any of these are out: wrong era, wrong register for a
# clarity hint, or a dialect this feature must recognize rather than
# "correct" (that is the dialect-protection layer's job, not this list's).
SKIP_TAGS = {
    "obsolete", "archaic", "historical", "offensive", "derogatory", "vulgar",
    "slur", "ethnic", "misspelling", "nonstandard", "eye-dialect",
    "African-American-Vernacular", "AAVE",
}
SKIP_LABEL_RE = re.compile(
    r"African[- ]American|AAVE|obsolete|archaic|historical|offensive|derogatory|vulgar",
    re.IGNORECASE)

REGISTER_TAGS = {"informal", "colloquial", "slang", "dated", "dialectal", "rural"}

ALLOWED_POS = {"noun", "verb", "adj", "adv", "intj", "phrase"}

GLOSS_ARTICLE_RE = re.compile(r"^(?:an?|the)\s+", re.IGNORECASE)


def sense_label_text(sense):
    parts = list(sense.get("tags", [])) + list(sense.get("raw_tags", []))
    for raw in sense.get("raw_glosses", [])[:1]:
        match = re.match(r"\(([^)]*)\)", raw)
        if match:
            parts.append(match.group(1))
    return " | ".join(str(part) for part in parts)


def regions_of(label_text):
    return sorted(code for code, pattern in REGION_RES.items() if pattern.search(label_text))


def gloss_alternative(gloss):
    """A short gloss doubles as the suggested alternative: 'A drinking
    fountain.' -> 'drinking fountain'."""
    text = GLOSS_ARTICLE_RE.sub("", gloss.strip().rstrip(".")).strip()
    if 0 < len(text.split()) <= 4 and re.fullmatch(r"[A-Za-z' -]+", text):
        return text.lower()
    return None


def load_english_words(dic_path):
    english = set()
    with open(dic_path, encoding="utf-8", errors="replace") as handle:
        for line in list(handle)[1:]:
            word = line.split("/")[0].strip().lower()
            if word:
                english.add(word)
    return english


def load_roster_tokens(roster_path):
    data = json.load(open(roster_path, encoding="utf-8"))
    tokens = set()
    for names in data.get("books", {}).values():
        for name in names:
            for token in re.findall(r"[A-Za-z']+", name):
                tokens.add(token.lower())
    return tokens


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("raw", help="raw-wiktextract-data.jsonl.gz")
    parser.add_argument("--dic", required=True, help="bundled en_US.dic for the name screen")
    parser.add_argument("--roster", required=True, help="publish/third-party-roster.json")
    parser.add_argument("--out", default=".", help="output directory")
    args = parser.parse_args()

    english = load_english_words(args.dic)
    roster_guard = load_roster_tokens(args.roster) - english

    candidates = {}
    common_words = set()  # words with at least one everyday, non-regional sense
    lines = 0

    opener = gzip.open if args.raw.endswith(".gz") else open
    with opener(args.raw, "rt", encoding="utf-8", errors="replace") as handle:
        for line in handle:
            lines += 1
            if lines % 1_000_000 == 0:
                print(f"  …{lines:,} lines, {len(candidates):,} candidates", file=sys.stderr)
            try:
                entry = json.loads(line)
            except json.JSONDecodeError:
                continue
            if entry.get("lang_code") != "en":
                continue
            word = entry.get("word", "")
            pos = entry.get("pos", "")
            if not word or pos not in ALLOWED_POS:
                continue
            if len(word) < 3 or len(word.split()) > 3:
                continue
            if word != word.lower():  # proper nouns and brands are not vocabulary
                continue

            entry_synonyms = [s.get("word", "") for s in entry.get("synonyms", [])]

            for sense in entry.get("senses", []):
                labels = sense_label_text(sense)
                tags = set(sense.get("tags", []))
                glosses = sense.get("glosses", [])
                if tags & SKIP_TAGS or SKIP_LABEL_RE.search(labels):
                    continue
                found = regions_of(labels)
                if not found:
                    # An untagged, current sense makes the word everyday
                    # vocabulary somewhere it is not regional.
                    if glosses and not (tags & {"rare", "dialectal"}):
                        common_words.add(word)
                    continue
                if not glosses:
                    continue

                meaning = glosses[0].strip()
                alternatives = [s.get("word", "") for s in sense.get("synonyms", [])]
                alternatives = [a for a in alternatives if a and a.lower() != word]
                if not alternatives:
                    alternatives = [a for a in entry_synonyms if a and a.lower() != word]
                if not alternatives:
                    derived = gloss_alternative(meaning)
                    if derived and derived != word:
                        alternatives = [derived]
                if not alternatives:
                    continue

                item = candidates.setdefault(word, {
                    "id": "regional." + word.replace(" ", "-"),
                    "forms": [word],
                    "pos": pos,
                    "category": "regional_vocabulary",
                    "regions": [],
                    "register": [],
                    "meaning": meaning[:160],
                    "alternatives": [],
                    "ambiguity": "low",
                    "dialogue_policy": "preserve",
                    "narration_policy": "suggest",
                    "auto_replace": False,
                    "source": "wiktionary",
                })
                item["regions"] = sorted(set(item["regions"]) | set(found))
                item["register"] = sorted(set(item["register"]) | (tags & REGISTER_TAGS))
                for alternative in alternatives[:4]:
                    if alternative.lower() not in [a.lower() for a in item["alternatives"]]:
                        item["alternatives"].append(alternative)

    screened = []
    kept = []
    for word, item in candidates.items():
        if word in common_words:
            item["ambiguity"] = "high"
        if any(token in roster_guard for token in word.split()):
            screened.append(word)
            continue
        kept.append(item)
    kept.sort(key=lambda item: (item["regions"][0], item["forms"][0]))

    out = pathlib.Path(args.out)
    out.mkdir(parents=True, exist_ok=True)
    with open(out / "candidates.json", "w", encoding="utf-8") as handle:
        json.dump(kept, handle, indent=1, ensure_ascii=False)

    by_region = defaultdict(list)
    for item in kept:
        by_region[item["regions"][0]].append(item)
    with open(out / "candidates.md", "w", encoding="utf-8") as handle:
        handle.write("# Regional vocabulary candidates\n\n")
        handle.write("Review by deletion is not needed: add any word to `rejected.txt` "
                     "(one per line) and the build step drops it. `(!)` marks a word "
                     "with a common everyday sense; the engine treats those as "
                     "context-dependent.\n")
        for region in sorted(by_region):
            handle.write(f"\n## {region} ({len(by_region[region])})\n\n")
            for item in by_region[region]:
                flag = " (!)" if item["ambiguity"] == "high" else ""
                others = [r for r in item["regions"][1:]]
                extra = f" [{', '.join(others)}]" if others else ""
                handle.write(f"- **{item['forms'][0]}**{flag}{extra} — {item['meaning']}"
                             f" → {', '.join(item['alternatives'])}\n")

    print(f"{lines:,} lines read")
    print(f"{len(kept):,} candidates kept across {len(by_region)} regions")
    print(f"{len(screened)} dropped by the name screen: {', '.join(sorted(screened)[:20])}")


if __name__ == "__main__":
    main()
