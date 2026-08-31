export namespace types {

	export class AIRewriteResult {
	    result: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new AIRewriteResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.result = source["result"];
	        this.error = source["error"];
	    }
	}
	export class StoryAnalysisObservation {
	    kind: string;
	    level: string;
	    title: string;
	    detail: string;
	    chapter_index?: number;

	    static createFrom(source: any = {}) {
	        return new StoryAnalysisObservation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.level = source["level"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.chapter_index = source["chapter_index"];
	    }
	}
	export class TermCount {
	    term: string;
	    count: number;

	    static createFrom(source: any = {}) {
	        return new TermCount(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.term = source["term"];
	        this.count = source["count"];
	    }
	}
	export class ChapterAnalysis {
	    chapter_id?: string;
	    chapter_index: number;
	    title: string;
	    word_count: number;
	    sentence_count: number;
	    paragraph_count: number;
	    scene_break_count: number;
	    dialogue_percent: number;
	    average_sentence_words: number;
	    average_paragraph_words: number;
	    reading_ease: number;
	    mean_grade_level: number;
	    mean_word_length: number;
	    short_sentence_percent: number;
	    long_sentence_percent: number;
	    verb_percent: number;
	    adverb_percent: number;
	    adjective_percent: number;
	    tempo_score: number;
	    tempo_label: string;
	    keywords?: TermCount[];
	    extractive_summary?: string;

	    static createFrom(source: any = {}) {
	        return new ChapterAnalysis(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter_id = source["chapter_id"];
	        this.chapter_index = source["chapter_index"];
	        this.title = source["title"];
	        this.word_count = source["word_count"];
	        this.sentence_count = source["sentence_count"];
	        this.paragraph_count = source["paragraph_count"];
	        this.scene_break_count = source["scene_break_count"];
	        this.dialogue_percent = source["dialogue_percent"];
	        this.average_sentence_words = source["average_sentence_words"];
	        this.average_paragraph_words = source["average_paragraph_words"];
	        this.reading_ease = source["reading_ease"];
	        this.mean_grade_level = source["mean_grade_level"];
	        this.mean_word_length = source["mean_word_length"];
	        this.short_sentence_percent = source["short_sentence_percent"];
	        this.long_sentence_percent = source["long_sentence_percent"];
	        this.verb_percent = source["verb_percent"];
	        this.adverb_percent = source["adverb_percent"];
	        this.adjective_percent = source["adjective_percent"];
	        this.tempo_score = source["tempo_score"];
	        this.tempo_label = source["tempo_label"];
	        this.keywords = this.convertValues(source["keywords"], TermCount);
	        this.extractive_summary = source["extractive_summary"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class StoryAnalysisOverview {
	    chapter_count: number;
	    word_count: number;
	    sentence_count: number;
	    paragraph_count: number;
	    average_chapter_words: number;
	    average_sentence_words: number;
	    dialogue_percent: number;
	    reading_ease: number;
	    mean_grade_level: number;
	    tempo_score: number;

	    static createFrom(source: any = {}) {
	        return new StoryAnalysisOverview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter_count = source["chapter_count"];
	        this.word_count = source["word_count"];
	        this.sentence_count = source["sentence_count"];
	        this.paragraph_count = source["paragraph_count"];
	        this.average_chapter_words = source["average_chapter_words"];
	        this.average_sentence_words = source["average_sentence_words"];
	        this.dialogue_percent = source["dialogue_percent"];
	        this.reading_ease = source["reading_ease"];
	        this.mean_grade_level = source["mean_grade_level"];
	        this.tempo_score = source["tempo_score"];
	    }
	}
	export class StoryAnalysisData {
	    content_hash: string;
	    engine: string;
	    last_analyzed: string;
	    overview: StoryAnalysisOverview;
	    chapters: ChapterAnalysis[];
	    observations?: StoryAnalysisObservation[];
	    version: number;

	    static createFrom(source: any = {}) {
	        return new StoryAnalysisData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content_hash = source["content_hash"];
	        this.engine = source["engine"];
	        this.last_analyzed = source["last_analyzed"];
	        this.overview = this.convertValues(source["overview"], StoryAnalysisOverview);
	        this.chapters = this.convertValues(source["chapters"], ChapterAnalysis);
	        this.observations = this.convertValues(source["observations"], StoryAnalysisObservation);
	        this.version = source["version"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CharacterEvent {
	    id: string;
	    character_ids: string[];
	    chapter_index: number;
	    event_type: string;
	    description: string;
	    is_auto_detected: boolean;

	    static createFrom(source: any = {}) {
	        return new CharacterEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.character_ids = source["character_ids"];
	        this.chapter_index = source["chapter_index"];
	        this.event_type = source["event_type"];
	        this.description = source["description"];
	        this.is_auto_detected = source["is_auto_detected"];
	    }
	}
	export class RelationshipRecord {
	    id: string;
	    character1_id: string;
	    character2_id: string;
	    first_chapter: number;
	    last_chapter: number;
	    interaction_count: number;
	    strength: number;
	    chapter_history: number[];
	    interaction_ids?: string[];
	    type_breakdown: Record<string, number>;

	    static createFrom(source: any = {}) {
	        return new RelationshipRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.character1_id = source["character1_id"];
	        this.character2_id = source["character2_id"];
	        this.first_chapter = source["first_chapter"];
	        this.last_chapter = source["last_chapter"];
	        this.interaction_count = source["interaction_count"];
	        this.strength = source["strength"];
	        this.chapter_history = source["chapter_history"];
	        this.interaction_ids = source["interaction_ids"];
	        this.type_breakdown = source["type_breakdown"];
	    }
	}
	export class InteractionRecord {
	    id: string;
	    participants: string[];
	    chapter_index: number;
	    scene_id?: string;
	    sentence_id?: string;
	    interaction_type: string;
	    directed_from?: string;
	    directed_to?: string;
	    confidence: number;
	    text_snippet?: string;

	    static createFrom(source: any = {}) {
	        return new InteractionRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.participants = source["participants"];
	        this.chapter_index = source["chapter_index"];
	        this.scene_id = source["scene_id"];
	        this.sentence_id = source["sentence_id"];
	        this.interaction_type = source["interaction_type"];
	        this.directed_from = source["directed_from"];
	        this.directed_to = source["directed_to"];
	        this.confidence = source["confidence"];
	        this.text_snippet = source["text_snippet"];
	    }
	}
	export class SceneRecord {
	    id: string;
	    chapter_index: number;
	    start_offset: number;
	    end_offset: number;
	    scene_type: string;
	    character_ids: string[];

	    static createFrom(source: any = {}) {
	        return new SceneRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.chapter_index = source["chapter_index"];
	        this.start_offset = source["start_offset"];
	        this.end_offset = source["end_offset"];
	        this.scene_type = source["scene_type"];
	        this.character_ids = source["character_ids"];
	    }
	}
	export class RelationshipData {
	    scenes?: SceneRecord[];
	    interactions?: InteractionRecord[];
	    relationships?: RelationshipRecord[];
	    events?: CharacterEvent[];
	    last_analyzed?: string;
	    version?: number;

	    static createFrom(source: any = {}) {
	        return new RelationshipData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scenes = this.convertValues(source["scenes"], SceneRecord);
	        this.interactions = this.convertValues(source["interactions"], InteractionRecord);
	        this.relationships = this.convertValues(source["relationships"], RelationshipRecord);
	        this.events = this.convertValues(source["events"], CharacterEvent);
	        this.last_analyzed = source["last_analyzed"];
	        this.version = source["version"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EntityDecision {
	    names: string[];
	    status: string;

	    static createFrom(source: any = {}) {
	        return new EntityDecision(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.names = source["names"];
	        this.status = source["status"];
	    }
	}
	export class MergeRule {
	    name_1: string;
	    name_2: string;

	    static createFrom(source: any = {}) {
	        return new MergeRule(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name_1 = source["name_1"];
	        this.name_2 = source["name_2"];
	    }
	}
	export class SeparatedPairRecord {
	    mention_id_1: string;
	    mention_id_2: string;
	    reason?: string;

	    static createFrom(source: any = {}) {
	        return new SeparatedPairRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mention_id_1 = source["mention_id_1"];
	        this.mention_id_2 = source["mention_id_2"];
	        this.reason = source["reason"];
	    }
	}
	export class EntityRecord {
	    id: string;
	    canonical: string;
	    aliases: string[];
	    mention_ids: string[];
	    confidence: number;
	    titles: string[];
	    kind?: string;
	    detection_status?: string;
	    detection_score?: number;
	    character_id?: string;

	    static createFrom(source: any = {}) {
	        return new EntityRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.canonical = source["canonical"];
	        this.aliases = source["aliases"];
	        this.mention_ids = source["mention_ids"];
	        this.confidence = source["confidence"];
	        this.titles = source["titles"];
	        this.kind = source["kind"];
	        this.detection_status = source["detection_status"];
	        this.detection_score = source["detection_score"];
	        this.character_id = source["character_id"];
	    }
	}
	export class MentionRecord {
	    id: string;
	    text: string;
	    sentence_id: string;
	    chapter: number;
	    char_offset: number;
	    person_evidence?: boolean;
	    non_person_evidence?: boolean;
	    strong_person_evidence?: boolean;

	    static createFrom(source: any = {}) {
	        return new MentionRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.sentence_id = source["sentence_id"];
	        this.chapter = source["chapter"];
	        this.char_offset = source["char_offset"];
	        this.person_evidence = source["person_evidence"];
	        this.non_person_evidence = source["non_person_evidence"];
	        this.strong_person_evidence = source["strong_person_evidence"];
	    }
	}
	export class EntityData {
	    mentions?: MentionRecord[];
	    entities?: EntityRecord[];
	    separated_pairs?: SeparatedPairRecord[];
	    merge_rules?: MergeRule[];
	    decisions?: EntityDecision[];
	    last_resolved?: string;
	    version?: number;

	    static createFrom(source: any = {}) {
	        return new EntityData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mentions = this.convertValues(source["mentions"], MentionRecord);
	        this.entities = this.convertValues(source["entities"], EntityRecord);
	        this.separated_pairs = this.convertValues(source["separated_pairs"], SeparatedPairRecord);
	        this.merge_rules = this.convertValues(source["merge_rules"], MergeRule);
	        this.decisions = this.convertValues(source["decisions"], EntityDecision);
	        this.last_resolved = source["last_resolved"];
	        this.version = source["version"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AnalysisData {
	    entity_resolution?: EntityData;
	    relationships?: RelationshipData;
	    story?: StoryAnalysisData;
	    version?: number;

	    static createFrom(source: any = {}) {
	        return new AnalysisData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entity_resolution = this.convertValues(source["entity_resolution"], EntityData);
	        this.relationships = this.convertValues(source["relationships"], RelationshipData);
	        this.story = this.convertValues(source["story"], StoryAnalysisData);
	        this.version = source["version"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppSettings {
	    default_author: string;
	    default_publisher: string;
	    default_copyright: string;
	    default_save_dir: string;
	    dark_mode: boolean;
	    theme_mode: string;
	    auto_theme_use_manual: boolean;
	    auto_theme_dawn: string;
	    auto_theme_dusk: string;
	    activity_autosave_enabled: boolean;
	    custom_dictionary?: string[];
	    spell_check_enabled: boolean;
	    grammar_check_enabled: boolean;
	    cast_enabled: boolean;
	    story_bible_enabled: boolean;
	    plot_walker_enabled: boolean;
	    analysis_enabled: boolean;
	    characters_lane_view: string;
	    ai_enabled: boolean;
	    ai_mode: string;
	    ai_provider: string;
	    ai_api_key?: string;
	    has_api_key: boolean;
	    ai_debug_logging: boolean;
	    ai_model: string;
	    ai_local_endpoint: string;
	    ai_local_model: string;
	    prose_guide: string;
	    book_font: string;
	    editor_font_size: string;
	    book_font_size: number;
	    book_line_spacing: string;
	    book_drop_caps: boolean;
	    book_trim_size: string;
	    sidebar_panel_width: number;
	    sidebar_active_section: string;

	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.default_author = source["default_author"];
	        this.default_publisher = source["default_publisher"];
	        this.default_copyright = source["default_copyright"];
	        this.default_save_dir = source["default_save_dir"];
	        this.dark_mode = source["dark_mode"];
	        this.theme_mode = source["theme_mode"];
	        this.auto_theme_use_manual = source["auto_theme_use_manual"];
	        this.auto_theme_dawn = source["auto_theme_dawn"];
	        this.auto_theme_dusk = source["auto_theme_dusk"];
	        this.activity_autosave_enabled = source["activity_autosave_enabled"];
	        this.custom_dictionary = source["custom_dictionary"];
	        this.spell_check_enabled = source["spell_check_enabled"];
	        this.grammar_check_enabled = source["grammar_check_enabled"];
	        this.cast_enabled = source["cast_enabled"];
	        this.story_bible_enabled = source["story_bible_enabled"];
	        this.plot_walker_enabled = source["plot_walker_enabled"];
	        this.analysis_enabled = source["analysis_enabled"];
	        this.characters_lane_view = source["characters_lane_view"];
	        this.ai_enabled = source["ai_enabled"];
	        this.ai_mode = source["ai_mode"];
	        this.ai_provider = source["ai_provider"];
	        this.ai_api_key = source["ai_api_key"];
	        this.has_api_key = source["has_api_key"];
	        this.ai_debug_logging = source["ai_debug_logging"];
	        this.ai_model = source["ai_model"];
	        this.ai_local_endpoint = source["ai_local_endpoint"];
	        this.ai_local_model = source["ai_local_model"];
	        this.prose_guide = source["prose_guide"];
	        this.book_font = source["book_font"];
	        this.editor_font_size = source["editor_font_size"];
	        this.book_font_size = source["book_font_size"];
	        this.book_line_spacing = source["book_line_spacing"];
	        this.book_drop_caps = source["book_drop_caps"];
	        this.book_trim_size = source["book_trim_size"];
	        this.sidebar_panel_width = source["sidebar_panel_width"];
	        this.sidebar_active_section = source["sidebar_active_section"];
	    }
	}
	export class BackupInfo {
	    number: number;
	    path: string;
	    modified: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new BackupInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.path = source["path"];
	        this.modified = source["modified"];
	        this.size = source["size"];
	    }
	}
	export class Beat {
	    id: string;
	    chapter_index: number;
	    beat_type: string;
	    description: string;
	    notes?: string;

	    static createFrom(source: any = {}) {
	        return new Beat(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.chapter_index = source["chapter_index"];
	        this.beat_type = source["beat_type"];
	        this.description = source["description"];
	        this.notes = source["notes"];
	    }
	}
	export class BeatSheet {
	    beats: Beat[];

	    static createFrom(source: any = {}) {
	        return new BeatSheet(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.beats = this.convertValues(source["beats"], Beat);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class KnowledgeEntry {
	    secret_id: string;
	    character_id: string;
	    learns_chapter?: number;
	    suspected_chapter?: number;

	    static createFrom(source: any = {}) {
	        return new KnowledgeEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.secret_id = source["secret_id"];
	        this.character_id = source["character_id"];
	        this.learns_chapter = source["learns_chapter"];
	        this.suspected_chapter = source["suspected_chapter"];
	    }
	}
	export class SecretInfo {
	    id: string;
	    name: string;
	    description: string;

	    static createFrom(source: any = {}) {
	        return new SecretInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class KnowledgeMatrix {
	    secrets: SecretInfo[];
	    entries: KnowledgeEntry[];

	    static createFrom(source: any = {}) {
	        return new KnowledgeMatrix(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.secrets = this.convertValues(source["secrets"], SecretInfo);
	        this.entries = this.convertValues(source["entries"], KnowledgeEntry);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ForeshadowingItem {
	    id: string;
	    name: string;
	    description: string;
	    plant_chapter: number;
	    reinforce_chapters?: number[];
	    payoff_chapter?: number;
	    status: string;
	    notes?: string;

	    static createFrom(source: any = {}) {
	        return new ForeshadowingItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.plant_chapter = source["plant_chapter"];
	        this.reinforce_chapters = source["reinforce_chapters"];
	        this.payoff_chapter = source["payoff_chapter"];
	        this.status = source["status"];
	        this.notes = source["notes"];
	    }
	}
	export class ForeshadowingLedger {
	    items: ForeshadowingItem[];

	    static createFrom(source: any = {}) {
	        return new ForeshadowingLedger(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ForeshadowingItem);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class WritingStyleOptions {
	    metaphors: number;
	    similes: number;
	    sensory_detail: number;
	    internal_thought: number;
	    dialogue: number;
	    action: number;
	    description: number;
	    pacing: number;

	    static createFrom(source: any = {}) {
	        return new WritingStyleOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.metaphors = source["metaphors"];
	        this.similes = source["similes"];
	        this.sensory_detail = source["sensory_detail"];
	        this.internal_thought = source["internal_thought"];
	        this.dialogue = source["dialogue"];
	        this.action = source["action"];
	        this.description = source["description"];
	        this.pacing = source["pacing"];
	    }
	}
	export class WritingGoals {
	    target_word_count: number;
	    daily_word_goal: number;
	    words_today: number;
	    last_writing_date: string;

	    static createFrom(source: any = {}) {
	        return new WritingGoals(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target_word_count = source["target_word_count"];
	        this.daily_word_goal = source["daily_word_goal"];
	        this.words_today = source["words_today"];
	        this.last_writing_date = source["last_writing_date"];
	    }
	}
	export class Character {
	    id: string;
	    name: string;
	    role: string;
	    description: string;
	    appearance: string;
	    personality: string;
	    motivation: string;
	    notes: string;
	    is_auto_detected?: boolean;
	    aliases?: string[];
	    mention_count?: number;
	    first_chapter?: number;
	    chapter_mentions?: Record<number, number>;
	    attributes?: Record<string, string>;
	    entity_kind?: string;
	    detection_status?: string;
	    detection_score?: number;

	    static createFrom(source: any = {}) {
	        return new Character(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.role = source["role"];
	        this.description = source["description"];
	        this.appearance = source["appearance"];
	        this.personality = source["personality"];
	        this.motivation = source["motivation"];
	        this.notes = source["notes"];
	        this.is_auto_detected = source["is_auto_detected"];
	        this.aliases = source["aliases"];
	        this.mention_count = source["mention_count"];
	        this.first_chapter = source["first_chapter"];
	        this.chapter_mentions = source["chapter_mentions"];
	        this.attributes = source["attributes"];
	        this.entity_kind = source["entity_kind"];
	        this.detection_status = source["detection_status"];
	        this.detection_score = source["detection_score"];
	    }
	}
	export class StoryBible {
	    characters: Character[];
	    plot_notes: string;
	    timeline: string;

	    static createFrom(source: any = {}) {
	        return new StoryBible(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.characters = this.convertValues(source["characters"], Character);
	        this.plot_notes = source["plot_notes"];
	        this.timeline = source["timeline"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChapterItem {
	    id?: string;
	    title: string;
	    subtitle?: string;
	    type: string;
	    content: string;

	    static createFrom(source: any = {}) {
	        return new ChapterItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.subtitle = source["subtitle"];
	        this.type = source["type"];
	        this.content = source["content"];
	    }
	}
	export class Metadata {
	    title: string;
	    author: string;
	    isbn: string;
	    publisher: string;
	    created: string;
	    modified: string;

	    static createFrom(source: any = {}) {
	        return new Metadata(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.author = source["author"];
	        this.isbn = source["isbn"];
	        this.publisher = source["publisher"];
	        this.created = source["created"];
	        this.modified = source["modified"];
	    }
	}
	export class BookData {
	    version: string;
	    metadata: Metadata;
	    copyright: string;
	    front_matter: ChapterItem[];
	    body: ChapterItem[];
	    back_matter: ChapterItem[];
	    file_path?: string;
	    story_bible?: StoryBible;
	    writing_goals?: WritingGoals;
	    style_options?: WritingStyleOptions;
	    is_indexed?: boolean;
	    last_indexed?: string;
	    beat_sheet?: BeatSheet;
	    foreshadowing?: ForeshadowingLedger;
	    knowledge_matrix?: KnowledgeMatrix;
	    analysis?: AnalysisData;

	    static createFrom(source: any = {}) {
	        return new BookData(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.metadata = this.convertValues(source["metadata"], Metadata);
	        this.copyright = source["copyright"];
	        this.front_matter = this.convertValues(source["front_matter"], ChapterItem);
	        this.body = this.convertValues(source["body"], ChapterItem);
	        this.back_matter = this.convertValues(source["back_matter"], ChapterItem);
	        this.file_path = source["file_path"];
	        this.story_bible = this.convertValues(source["story_bible"], StoryBible);
	        this.writing_goals = this.convertValues(source["writing_goals"], WritingGoals);
	        this.style_options = this.convertValues(source["style_options"], WritingStyleOptions);
	        this.is_indexed = source["is_indexed"];
	        this.last_indexed = source["last_indexed"];
	        this.beat_sheet = this.convertValues(source["beat_sheet"], BeatSheet);
	        this.foreshadowing = this.convertValues(source["foreshadowing"], ForeshadowingLedger);
	        this.knowledge_matrix = this.convertValues(source["knowledge_matrix"], KnowledgeMatrix);
	        this.analysis = this.convertValues(source["analysis"], AnalysisData);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ChapterHistoryEntry {
	    id: string;
	    chapter_id: string;
	    section: string;
	    chapter_title: string;
	    created_at: string;
	    reason: string;
	    word_count: number;
	    content_hash: string;
	    file: string;

	    static createFrom(source: any = {}) {
	        return new ChapterHistoryEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.chapter_id = source["chapter_id"];
	        this.section = source["section"];
	        this.chapter_title = source["chapter_title"];
	        this.created_at = source["created_at"];
	        this.reason = source["reason"];
	        this.word_count = source["word_count"];
	        this.content_hash = source["content_hash"];
	        this.file = source["file"];
	    }
	}
	export class ChapterHistorySnapshot {
	    entry: ChapterHistoryEntry;
	    content: string;

	    static createFrom(source: any = {}) {
	        return new ChapterHistorySnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = this.convertValues(source["entry"], ChapterHistoryEntry);
	        this.content = source["content"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ChapterSnapshotRequest {
	    chapter_id: string;
	    section: string;
	    chapter_title: string;
	    content: string;
	    reason?: string;

	    static createFrom(source: any = {}) {
	        return new ChapterSnapshotRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter_id = source["chapter_id"];
	        this.section = source["section"];
	        this.chapter_title = source["chapter_title"];
	        this.content = source["content"];
	        this.reason = source["reason"];
	    }
	}


	export class CharacterTimelineEvent {
	    chapter: number;
	    event_type: string;
	    description: string;
	    related_chars?: string[];

	    static createFrom(source: any = {}) {
	        return new CharacterTimelineEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter = source["chapter"];
	        this.event_type = source["event_type"];
	        this.description = source["description"];
	        this.related_chars = source["related_chars"];
	    }
	}
	export class CharacterTimelineResult {
	    success: boolean;
	    error?: string;
	    character_id: string;
	    events: CharacterTimelineEvent[];

	    static createFrom(source: any = {}) {
	        return new CharacterTimelineResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.character_id = source["character_id"];
	        this.events = this.convertValues(source["events"], CharacterTimelineEvent);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ClaudeCodeStatus {
	    installed: boolean;
	    authenticated: boolean;
	    npm_available: boolean;
	    version: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new ClaudeCodeStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.authenticated = source["authenticated"];
	        this.npm_available = source["npm_available"];
	        this.version = source["version"];
	        this.error = source["error"];
	    }
	}



	export class ExportOptions {
	    includeCopyright: boolean;
	    includeFrontMatter: boolean;
	    includeBackMatter: boolean;

	    static createFrom(source: any = {}) {
	        return new ExportOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeCopyright = source["includeCopyright"];
	        this.includeFrontMatter = source["includeFrontMatter"];
	        this.includeBackMatter = source["includeBackMatter"];
	    }
	}
	export class ExportResult {
	    success: boolean;
	    file_path?: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.file_path = source["file_path"];
	        this.error = source["error"];
	    }
	}


	export class FullAnalysisResult {
	    success: boolean;
	    error?: string;
	    book?: BookData;

	    static createFrom(source: any = {}) {
	        return new FullAnalysisResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.book = this.convertValues(source["book"], BookData);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ImportResult {
	    success: boolean;
	    book?: BookData;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.book = this.convertValues(source["book"], BookData);
	        this.error = source["error"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IndexResult {
	    success: boolean;
	    error?: string;
	    characters_found: number;
	    new_characters: number;
	    characters?: Character[];
	    book: BookData;

	    static createFrom(source: any = {}) {
	        return new IndexResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.characters_found = source["characters_found"];
	        this.new_characters = source["new_characters"];
	        this.characters = this.convertValues(source["characters"], Character);
	        this.book = this.convertValues(source["book"], BookData);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InlineGenerateRequest {
	    instruction: string;
	    before_context: string;
	    after_context: string;
	    characters: string[];
	    chapter_title: string;

	    static createFrom(source: any = {}) {
	        return new InlineGenerateRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.instruction = source["instruction"];
	        this.before_context = source["before_context"];
	        this.after_context = source["after_context"];
	        this.characters = source["characters"];
	        this.chapter_title = source["chapter_title"];
	    }
	}






	export class PDFOptions {
	    includeCopyright: boolean;
	    includeFrontMatter: boolean;
	    includeBackMatter: boolean;
	    pageSize: string;
	    fontSize: number;

	    static createFrom(source: any = {}) {
	        return new PDFOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeCopyright = source["includeCopyright"];
	        this.includeFrontMatter = source["includeFrontMatter"];
	        this.includeBackMatter = source["includeBackMatter"];
	        this.pageSize = source["pageSize"];
	        this.fontSize = source["fontSize"];
	    }
	}
	export class PrintPDFOptions {
	    includeCopyright: boolean;
	    includeFrontMatter: boolean;
	    includeBackMatter: boolean;
	    pageSize: string;
	    fontSize: number;
	    trimSize: string;
	    customWidth: string;
	    customHeight: string;
	    bleed: string;
	    gutterMargin: string;
	    outerMargin: string;
	    topMargin: string;
	    bottomMargin: string;
	    includeCropMarks: boolean;
	    fontFamily: string;
	    lineHeight: number;
	    paragraphIndent: string;
	    textAlign: string;
	    chapterStartsRecto: boolean;
	    dropCap: boolean;
	    dropCapLines: number;
	    runningHeaders: boolean;
	    headerStyle: string;
	    pageNumberPosition: string;
	    generateHalfTitle: boolean;
	    generateTOC: boolean;
	    mirroredMargins: boolean;

	    static createFrom(source: any = {}) {
	        return new PrintPDFOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeCopyright = source["includeCopyright"];
	        this.includeFrontMatter = source["includeFrontMatter"];
	        this.includeBackMatter = source["includeBackMatter"];
	        this.pageSize = source["pageSize"];
	        this.fontSize = source["fontSize"];
	        this.trimSize = source["trimSize"];
	        this.customWidth = source["customWidth"];
	        this.customHeight = source["customHeight"];
	        this.bleed = source["bleed"];
	        this.gutterMargin = source["gutterMargin"];
	        this.outerMargin = source["outerMargin"];
	        this.topMargin = source["topMargin"];
	        this.bottomMargin = source["bottomMargin"];
	        this.includeCropMarks = source["includeCropMarks"];
	        this.fontFamily = source["fontFamily"];
	        this.lineHeight = source["lineHeight"];
	        this.paragraphIndent = source["paragraphIndent"];
	        this.textAlign = source["textAlign"];
	        this.chapterStartsRecto = source["chapterStartsRecto"];
	        this.dropCap = source["dropCap"];
	        this.dropCapLines = source["dropCapLines"];
	        this.runningHeaders = source["runningHeaders"];
	        this.headerStyle = source["headerStyle"];
	        this.pageNumberPosition = source["pageNumberPosition"];
	        this.generateHalfTitle = source["generateHalfTitle"];
	        this.generateTOC = source["generateTOC"];
	        this.mirroredMargins = source["mirroredMargins"];
	    }
	}
	export class RecentProjectStats {
	    books?: number;
	    chapters: number;
	    words: number;

	    static createFrom(source: any = {}) {
	        return new RecentProjectStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.books = source["books"];
	        this.chapters = source["chapters"];
	        this.words = source["words"];
	    }
	}
	export class RecentProject {
	    type: string;
	    path: string;
	    name: string;
	    lastOpened: string;
	    stats: RecentProjectStats;

	    static createFrom(source: any = {}) {
	        return new RecentProject(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.lastOpened = source["lastOpened"];
	        this.stats = this.convertValues(source["stats"], RecentProjectStats);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class RelationshipAnalysisResult {
	    success: boolean;
	    error?: string;
	    book?: BookData;
	    scenes_detected: number;
	    interactions_found: number;
	    relationships_built: number;

	    static createFrom(source: any = {}) {
	        return new RelationshipAnalysisResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.book = this.convertValues(source["book"], BookData);
	        this.scenes_detected = source["scenes_detected"];
	        this.interactions_found = source["interactions_found"];
	        this.relationships_built = source["relationships_built"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}


	export class SaveResult {
	    success: boolean;
	    file_path: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new SaveResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.file_path = source["file_path"];
	        this.error = source["error"];
	    }
	}



	export class SplitEntityResult {
	    success: boolean;
	    error?: string;
	    book?: BookData;
	    characters?: Character[];

	    static createFrom(source: any = {}) {
	        return new SplitEntityResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.book = this.convertValues(source["book"], BookData);
	        this.characters = this.convertValues(source["characters"], Character);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}







}
