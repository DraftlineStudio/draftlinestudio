// Package indexing provides character detection and attribute extraction.
package indexing

// CommonWords contains words to exclude from character detection.
var CommonWords = map[string]bool{
	// Short words (1-2 chars) - filtered by regex but kept for safety
	"A": true, "I": true, "An": true, "He": true, "It": true, "We": true, "Or": true,
	"So": true, "No": true, "If": true, "My": true, "Me": true, "Us": true, "Up": true,
	// Pronouns and determiners (3+ chars)
	"The": true, "She": true, "You": true, "His": true, "Her": true, "Its": true,
	"Our": true, "They": true, "Your": true, "This": true, "That": true, "Him": true,
	"Their": true, "Mine": true, "Yours": true, "Ours": true, "Theirs": true,
	// Question words
	"Who": true, "What": true, "Where": true, "When": true, "Why": true, "How": true,
	// Common verbs and short words
	"Was": true, "Were": true, "Has": true, "Had": true, "Can": true, "Did": true,
	"Get": true, "Got": true, "Let": true, "See": true, "Say": true, "Use": true,
	// Conjunctions and prepositions
	"And": true, "But": true, "For": true, "Nor": true, "Yet": true, "Not": true,
	"All": true, "Any": true, "Out": true, "Now": true, "New": true, "Old": true,
	"Upon": true, "With": true, "From": true, "Over": true, "Under": true, "Along": true,
	// Adverbs and time words
	"There": true, "Here": true, "Then": true, "Just": true, "Only": true, "Even": true,
	"Still": true, "Already": true, "Very": true, "Really": true, "Quite": true, "Rather": true,
	"After": true, "Before": true, "During": true, "While": true, "About": true,
	"Against": true, "Between": true, "Into": true, "Through": true,
	"However": true, "Therefore": true, "Meanwhile": true, "Finally": true, "Suddenly": true,
	"Because": true, "Yes": true, "Maybe": true, "Perhaps": true,
	// Numbers
	"One": true, "Two": true, "Three": true, "Four": true, "Five": true,
	"First": true, "Second": true, "Third": true, "Last": true, "Next": true,
	"Few": true, "Way": true, "Day": true, "Man": true, "Boy": true, "Too": true, "Ago": true,
	// Titles (filtered but could appear in dialogue attribution)
	"Mr": true, "Mrs": true, "Ms": true, "Dr": true, "Sir": true, "Lord": true, "Lady": true,
	// Document/chapter words
	"Chapter": true, "Part": true, "Book": true, "Volume": true,
	// Days and months
	"Monday": true, "Tuesday": true, "Wednesday": true, "Thursday": true, "Friday": true, "Saturday": true, "Sunday": true,
	"January": true, "February": true, "March": true, "April": true, "May": true, "June": true,
	"July": true, "August": true, "September": true, "October": true, "November": true, "December": true,
	// Directions and places
	"North": true, "South": true, "East": true, "West": true,
	"God": true, "Earth": true, "Heaven": true, "Hell": true,
	// Indefinite pronouns
	"Something": true, "Nothing": true, "Everything": true, "Anything": true,
	"Someone": true, "Anyone": true, "Everyone": true,
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

// DialogueVerbs contains verbs used for dialogue attribution.
const DialogueVerbs = `said|asked|replied|answered|whispered|shouted|yelled|muttered|exclaimed|cried|called|screamed|murmured|snapped|growled|laughed|sighed|groaned|demanded|insisted|suggested|agreed|admitted|explained|continued|added|interrupted|announced|declared|observed|remarked|noted|commented|wondered|mused|thought|began|finished|concluded`
