// Package indexing provides character detection and attribute extraction.
package indexing

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// commonWordsLower contains words to exclude from character detection (all lowercase for case-insensitive matching).
var commonWordsLower = map[string]bool{
	// Short words (1-2 chars)
	"a": true, "i": true, "an": true, "he": true, "it": true, "we": true, "or": true,
	"so": true, "no": true, "if": true, "my": true, "me": true, "us": true, "up": true,
	"at": true, "as": true, "is": true, "in": true, "on": true, "of": true, "to": true,
	"by": true, "be": true, "am": true, "do": true, "go": true, "ah": true, "oh": true,
	"are": true, "been": true, "does": true, "said": true, "saying": true,
	// Pronouns and determiners (3+ chars)
	"the": true, "she": true, "you": true, "his": true, "her": true, "its": true,
	"our": true, "they": true, "your": true, "this": true, "that": true, "him": true,
	"their": true, "mine": true, "yours": true, "ours": true, "theirs": true,
	"each": true, "every": true, "some": true, "many": true,
	// Question words
	"who": true, "what": true, "where": true, "when": true, "why": true, "how": true,
	// Common verbs and short words
	"was": true, "were": true, "has": true, "had": true, "can": true, "did": true,
	"could": true, "would": true, "should": true, "might": true, "must": true, "shall": true,
	"get": true, "got": true, "let": true, "see": true, "say": true, "use": true,
	"lets": true, "let's": true, "says": true, "goes": true, "went": true, "come": true,
	"came": true, "take": true, "took": true, "make": true, "made": true, "give": true,
	"gave": true, "know": true, "knew": true, "think": true, "thought": true,
	"thank": true, "thanks": true, "carried": true, "remembers": true, "gives": true, "takes": true,
	"feel": true, "felt": true, "look": true, "looked": true, "seem": true, "seemed": true,
	"want": true, "wanted": true, "need": true, "needed": true, "like": true, "liked": true,
	"ask": true, "asked": true, "tell": true, "told": true, "call": true, "called": true,
	"try": true, "tried": true, "leave": true, "left": true, "keep": true, "kept": true,
	"begin": true, "began": true, "start": true, "started": true, "run": true, "ran": true,
	"show": true, "showed": true, "hear": true, "heard": true, "play": true, "played": true,
	"move": true, "moved": true, "live": true, "lived": true, "believe": true, "believed": true,
	"hold": true, "held": true, "bring": true, "brought": true, "happen": true, "happened": true,
	"write": true, "wrote": true, "stand": true, "stood": true, "sit": true, "sat": true,
	"lose": true, "lost": true, "pay": true, "paid": true, "meet": true, "met": true,
	"include": true, "included": true, "continue": true, "continued": true,
	"set": true, "learn": true, "learned": true, "change": true, "changed": true,
	"lead": true, "led": true, "understand": true, "understood": true,
	"watch": true, "watched": true, "follow": true, "followed": true,
	"stop": true, "stopped": true, "create": true, "created": true,
	"speak": true, "spoke": true, "read": true, "allow": true, "allowed": true,
	"add": true, "added": true, "spend": true, "spent": true, "grow": true, "grew": true,
	"open": true, "opened": true, "walk": true, "walked": true, "win": true, "won": true,
	"offer": true, "offered": true, "remember": true, "remembered": true,
	"love": true, "loved": true, "consider": true, "considered": true,
	"appear": true, "appeared": true, "buy": true, "bought": true,
	"wait": true, "waited": true, "serve": true, "served": true,
	"die": true, "died": true, "send": true, "sent": true, "expect": true, "expected": true,
	"build": true, "built": true, "stay": true, "stayed": true,
	"fall": true, "fell": true, "cut": true, "reach": true, "reached": true,
	"kill": true, "killed": true, "remain": true, "remained": true,
	"suggest": true, "suggested": true, "raise": true, "raised": true,
	"pass": true, "passed": true, "sell": true, "sold": true,
	"require": true, "required": true, "report": true, "reported": true,
	"decide": true, "decided": true, "pull": true, "pulled": true,
	// Conjunctions and prepositions
	"and": true, "but": true, "for": true, "nor": true, "yet": true, "not": true,
	"all": true, "any": true, "out": true, "now": true, "new": true, "old": true,
	"upon": true, "with": true, "from": true, "over": true, "under": true, "along": true,
	"behind": true, "beside": true, "besides": true, "beneath": true, "above": true,
	"below": true, "beyond": true, "across": true, "toward": true, "towards": true,
	"near": true, "nearby": true, "despite": true, "unless": true, "instead": true,
	"further": true, "moreover": true, "nevertheless": true, "nonetheless": true,
	"elsewhere": true, "anyway": true, "anyhow": true, "whoever": true,
	"whenever": true, "wherever": true, "whatever": true, "whichever": true,
	// Sentence-lead adjectives that glue onto names ("poor" already listed)
	"dear": true, "sweet": true, "silly": true, "brave": true,
	// Adverbs and time words
	"there": true, "here": true, "then": true, "just": true, "only": true, "even": true,
	"still": true, "already": true, "very": true, "really": true, "quite": true, "rather": true,
	"after": true, "before": true, "during": true, "while": true, "about": true,
	"against": true, "between": true, "into": true, "through": true,
	"however": true, "therefore": true, "meanwhile": true, "finally": true, "suddenly": true,
	"because": true, "yes": true, "maybe": true, "perhaps": true,
	"again": true, "never": true, "always": true, "often": true, "usually": true,
	"sometimes": true, "soon": true, "later": true, "early": true, "late": true,
	"away": true, "back": true, "down": true, "off": true, "also": true, "well": true,
	"much": true, "more": true, "most": true, "less": true, "least": true,
	"enough": true, "almost": true, "actually": true,
	// Numbers
	"one": true, "two": true, "three": true, "four": true, "five": true,
	"six": true, "seven": true, "eight": true, "nine": true, "ten": true,
	"first": true, "second": true, "third": true, "last": true, "next": true,
	"few": true, "way": true, "day": true, "man": true, "boy": true, "too": true, "ago": true,
	// Common nouns that appear capitalized at sentence start
	"manager": true, "director": true, "boss": true, "worker": true, "employee": true,
	"doctor": true, "nurse": true, "teacher": true, "student": true, "professor": true,
	"president": true, "chairman": true, "leader": true, "member": true, "owner": true,
	"driver": true, "pilot": true, "engineer": true, "lawyer": true, "judge": true,
	"officer": true, "soldier": true, "guard": true, "agent": true, "detective": true,
	"police": true, "cop": true, "chief": true, "captain": true, "sergeant": true,
	"lieutenant": true, "colonel": true, "general": true, "major": true, "commander": true,
	"clerk": true, "secretary": true, "assistant": true, "deputy": true, "aide": true,
	"waiter": true, "waitress": true, "chef": true, "cook": true, "baker": true,
	"writer": true, "author": true, "reporter": true, "journalist": true, "editor": true,
	"artist": true, "musician": true, "singer": true, "actor": true, "actress": true,
	"player": true, "coach": true, "trainer": true, "athlete": true, "champion": true,
	"friend": true, "enemy": true, "stranger": true, "visitor": true, "guest": true,
	"neighbor": true, "neighbour": true, "partner": true, "colleague": true, "associate": true,
	"woman": true, "girl": true, "guy": true, "fellow": true, "gentleman": true,
	"lady": true, "kid": true, "child": true, "baby": true, "teen": true, "teenager": true,
	"adult": true, "elder": true, "senior": true, "junior": true, "young": true,
	"people": true, "person": true, "human": true, "being": true, "creature": true,
	"thing": true, "stuff": true, "matter": true, "issue": true, "problem": true,
	"question": true, "answer": true, "reason": true, "cause": true, "effect": true,
	"result": true, "fact": true, "truth": true, "real": true, "reality": true,
	"life": true, "death": true, "world": true, "place": true, "home": true, "house": true,
	"room": true, "door": true, "window": true, "floor": true, "wall": true, "table": true,
	"chair": true, "bed": true, "desk": true, "office": true, "building": true,
	"street": true, "road": true, "city": true, "town": true, "village": true, "country": true,
	"state": true, "nation": true, "land": true, "area": true, "region": true, "zone": true,
	"side": true, "end": true, "part": true, "half": true, "whole": true, "rest": true,
	"top": true, "bottom": true, "front": true, "middle": true, "center": true, "centre": true,
	"right": true, "wrong": true, "good": true, "bad": true, "best": true, "worst": true,
	"great": true, "small": true, "big": true, "large": true, "little": true, "huge": true,
	"long": true, "short": true, "high": true, "low": true, "deep": true, "wide": true,
	"full": true, "empty": true, "close": true, "closed": true, "clear": true,
	"dark": true, "light": true, "bright": true, "black": true, "white": true, "red": true,
	"blue": true, "green": true, "yellow": true, "brown": true, "gray": true, "grey": true,
	"hot": true, "cold": true, "warm": true, "cool": true, "hard": true, "soft": true,
	"fast": true, "slow": true, "quick": true, "easy": true, "difficult": true, "simple": true,
	"same": true, "different": true, "other": true, "another": true, "own": true, "certain": true,
	"sure": true, "true": true, "false": true, "possible": true, "impossible": true,
	"able": true, "unable": true, "ready": true, "busy": true, "free": true, "safe": true,
	"sorry": true, "happy": true, "sad": true, "angry": true, "afraid": true, "alone": true,
	"alive": true, "dead": true, "sick": true, "healthy": true, "strong": true, "weak": true,
	"rich": true, "poor": true, "important": true, "special": true, "nice": true,
	"fine": true, "okay": true, "perfect": true, "exactly": true, "indeed": true,
	// Titles (filtered but could appear in dialogue attribution)
	"mr": true, "mrs": true, "ms": true, "dr": true, "sir": true, "lord": true,
	// Document/chapter words
	"chapter": true, "book": true, "volume": true, "page": true, "scene": true,
	"act": true, "episode": true, "section": true, "paragraph": true, "sentence": true,
	// Days and months
	"monday": true, "tuesday": true, "wednesday": true, "thursday": true, "friday": true, "saturday": true, "sunday": true,
	"january": true, "february": true, "march": true, "april": true, "may": true, "june": true,
	"july": true, "august": true, "september": true, "october": true, "november": true, "december": true,
	// Directions and places
	"north": true, "south": true, "east": true, "west": true,
	"god": true, "earth": true, "heaven": true, "hell": true,
	// Indefinite pronouns
	"something": true, "nothing": true, "everything": true, "anything": true,
	"someone": true, "anyone": true, "everyone": true, "nobody": true, "everybody": true,
	"somewhere": true, "anywhere": true, "everywhere": true, "nowhere": true,
	// Time-related
	"time": true, "moment": true, "minute": true, "hour": true, "week": true, "month": true, "year": true,
	"today": true, "tonight": true, "tomorrow": true, "yesterday": true,
	"morning": true, "afternoon": true, "evening": true, "night": true, "midnight": true, "noon": true,
	// Body parts (often appear capitalized)
	"head": true, "face": true, "eye": true, "eyes": true, "ear": true, "ears": true,
	"nose": true, "mouth": true, "hand": true, "hands": true, "arm": true, "arms": true,
	"leg": true, "legs": true, "foot": true, "feet": true, "body": true, "heart": true,
	"mind": true, "brain": true, "blood": true, "skin": true, "hair": true, "finger": true,
	// Common story words
	"story": true, "tale": true, "novel": true, "plot": true, "character": true,
	"hero": true, "villain": true, "narrator": true,
	"beginning": true, "ending": true, "climax": true,
}

// CommonWords wraps commonWordsLower for case-insensitive lookup
var CommonWords = commonWordsLower

// IsCommonWord checks if a word is a common word (case-insensitive)
func IsCommonWord(word string) bool {
	return commonWordsLower[strings.ToLower(word)]
}

// TemporalWords contains words that are clearly temporal/common.
var TemporalWords = map[string]bool{
	"today": true, "tomorrow": true, "yesterday": true,
	"morning": true, "evening": true, "afternoon": true, "tonight": true,
	"sometimes": true, "always": true, "never": true, "often": true,
	"perhaps": true, "maybe": true, "probably": true, "certainly": true,
	"however": true, "therefore": true, "although": true, "because": true,
	"before": true, "after": true, "during": true, "until": true,
	"inside": true, "outside": true, "between": true, "within": true,
	"around": true, "through": true, "without": true, "against": true,
}

// CommonWordSuffixes contains suffixes that are almost never names.
var CommonWordSuffixes = []string{
	"ly",    // adverbs: agreeably, quickly, suddenly
	"ing",   // gerunds: running, being, having (but not names like Ming)
	"tion",  // nouns: action, motion, station
	"sion",  // nouns: tension, mission
	"ness",  // nouns: darkness, happiness
	"ment",  // nouns: moment, movement
	"able",  // adjectives: capable, notable
	"ible",  // adjectives: possible, visible
	"ful",   // adjectives: beautiful, careful
	"less",  // adjectives: careless, hopeless
	"ous",   // adjectives: curious, nervous
	"ive",   // adjectives: active, creative
	"ward",  // directions: forward, backward
	"wards", // directions: afterwards, towards
	"wise",  // manner: otherwise, likewise
}

// FalsePositiveContextWords are words that, when following a name, suggest it's NOT a character.
// For example: "Hubbard Street" → Street suggests Hubbard is a place name, not a person.
// "General Tso's Chicken" → Chicken suggests this is a food reference.
var FalsePositiveContextWords = map[string]bool{
	// Street/location indicators
	"street": true, "avenue": true, "road": true, "boulevard": true, "drive": true,
	"lane": true, "way": true, "place": true, "court": true, "circle": true,
	"square": true, "park": true, "bridge": true, "tunnel": true, "highway": true,
	"building": true, "tower": true, "plaza": true, "center": true, "centre": true,
	"station": true, "airport": true, "hotel": true, "hospital": true, "school": true,
	"university": true, "college": true, "museum": true, "library": true, "theater": true,
	"theatre": true, "hall": true, "house": true, "manor": true, "castle": true,
	// Food indicators
	"chicken": true, "beef": true, "pork": true, "shrimp": true, "fish": true,
	"rice": true, "noodles": true, "sauce": true, "soup": true, "salad": true,
	"sandwich": true, "burger": true, "pizza": true, "steak": true, "lobster": true,
	// Brand/company/institution indicators
	"company": true, "corporation": true, "incorporated": true, "inc": true,
	"limited": true, "ltd": true, "industries": true, "enterprises": true,
	"foundation": true, "institute": true, "association": true, "organization": true,
	"department": true, "departments": true, "bureau": true, "division": true,
	"precinct": true, "headquarters": true, "agency": true, "ministry": true,
	"council": true, "committee": true, "unit": true, "squad": true,
	"academy": true, "corps": true, "patrol": true, "office": true, "offices": true,
	"facility": true, "laboratory": true,
	// Floor/location designations
	"level": true, "levels": true, "floor": true, "floors": true, "deck": true,
	"wing": true, "sector": true, "basement": true, "sublevel": true,
	// Geographic features
	"mountain": true, "river": true, "lake": true, "ocean": true, "sea": true,
	"valley": true, "canyon": true, "forest": true, "island": true, "peninsula": true,
	"desert": true, "bay": true, "harbor": true, "harbour": true, "creek": true,
	// Time/event indicators
	"day": true, "week": true, "month": true, "year": true, "era": true,
	"period": true, "age": true, "holiday": true, "festival": true,
	// Awards/titles as objects
	"award": true, "prize": true, "medal": true, "trophy": true,
}

// IsFalsePositiveContext checks if the text following a potential name suggests
// it's not a character name (e.g., "Hubbard Street", "General Tso's Chicken").
func IsFalsePositiveContext(text string, nameEndIndex int) bool {
	if nameEndIndex >= len(text) {
		return false
	}

	// Get the word(s) following the name
	remaining := text[nameEndIndex:]
	remaining = strings.TrimLeft(remaining, " \t")

	// Handle possessive "'s" followed by context word
	if strings.HasPrefix(remaining, "'s ") || strings.HasPrefix(remaining, "'s\t") {
		remaining = remaining[3:]
		remaining = strings.TrimLeft(remaining, " \t")
	}

	// Extract the next word
	nextWord := ""
	for i, r := range remaining {
		if r == ' ' || r == '\t' || r == ',' || r == '.' || r == '!' || r == '?' || r == '\n' {
			nextWord = remaining[:i]
			break
		}
		if i == len(remaining)-1 {
			nextWord = remaining
		}
	}

	if nextWord == "" {
		return false
	}

	return FalsePositiveContextWords[strings.ToLower(nextWord)]
}

var streetDesignators = map[string]bool{
	"st": true, "ave": true, "rd": true, "blvd": true, "dr": true,
	"ln": true, "ct": true, "pkwy": true, "hwy": true, "pl": true,
	"street": true, "avenue": true, "road": true, "boulevard": true,
	"drive": true, "lane": true, "court": true, "parkway": true,
	"highway": true, "place": true,
}

var locationPrepositions = map[string]bool{
	"at": true, "on": true, "onto": true, "along": true, "down": true,
	"from": true, "toward": true, "towards": true, "near": true,
	"off": true, "across": true, "through": true,
}

// IsAddressIntersectionContext catches constructions that generic NER often
// mistakes for people, such as "onto Ontario at Wells Ave". Requiring both a
// movement/location preposition before the candidate and a street-designated
// cross street after it avoids suppressing a person in "met Daniel at Wells".
func IsAddressIntersectionContext(text string, candidateStart, candidateEnd int) bool {
	if !locationPrepositions[previousWordLower(text, candidateStart)] {
		return false
	}

	after := strings.TrimLeft(text[candidateEnd:], " \t")
	lowerAfter := strings.ToLower(after)
	for _, connector := range []string{"at ", "and ", "& "} {
		if strings.HasPrefix(lowerAfter, connector) {
			after = after[len(connector):]
			words := scanWords(after)
			for i := 0; i < len(words) && i < 4; i++ {
				if streetDesignators[strings.ToLower(words[i].base)] {
					return true
				}
			}
			return false
		}
	}
	return false
}

func previousWordLower(text string, offset int) string {
	end := offset
	for end > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:end])
		if unicode.IsLetter(r) {
			break
		}
		end -= size
	}
	start := end
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(text[:start])
		if !unicode.IsLetter(r) {
			break
		}
		start -= size
	}
	return strings.ToLower(text[start:end])
}

func IsStreetDesignator(word string) bool {
	return streetDesignators[strings.ToLower(strings.TrimSuffix(word, "."))]
}
