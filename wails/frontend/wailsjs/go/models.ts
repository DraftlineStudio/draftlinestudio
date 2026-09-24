export namespace main {
	
	export class PluginInfo {
	    id: string;
	    name: string;
	    version: string;
	    publisher: string;
	    description: string;
	    api_version: number;
	    supported: boolean;
	    permissions: string[];
	    activation: string[];
	    contributes: plugins.Contributions;
	    has_frontend: boolean;
	    frontend_url: string;
	    has_sidecar: boolean;
	    running: boolean;
	    root: string;
	    load_error: string;
	
	    static createFrom(source: any = {}) {
	        return new PluginInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.publisher = source["publisher"];
	        this.description = source["description"];
	        this.api_version = source["api_version"];
	        this.supported = source["supported"];
	        this.permissions = source["permissions"];
	        this.activation = source["activation"];
	        this.contributes = this.convertValues(source["contributes"], plugins.Contributions);
	        this.has_frontend = source["has_frontend"];
	        this.frontend_url = source["frontend_url"];
	        this.has_sidecar = source["has_sidecar"];
	        this.running = source["running"];
	        this.root = source["root"];
	        this.load_error = source["load_error"];
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
	export class UpdateCheckResult {
	    current_version: string;
	    latest_version?: string;
	    latest_label?: string;
	    update_available: boolean;
	    release_url?: string;
	    release_notes?: string;
	    asset_name?: string;
	    asset_size?: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current_version = source["current_version"];
	        this.latest_version = source["latest_version"];
	        this.latest_label = source["latest_label"];
	        this.update_available = source["update_available"];
	        this.release_url = source["release_url"];
	        this.release_notes = source["release_notes"];
	        this.asset_name = source["asset_name"];
	        this.asset_size = source["asset_size"];
	        this.error = source["error"];
	    }
	}
	export class UpdateDownloadResult {
	    path?: string;
	    launched: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateDownloadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.launched = source["launched"];
	        this.error = source["error"];
	    }
	}

}

export namespace plugins {
	
	export class CommandContribution {
	    id: string;
	    shortcut?: string;
	
	    static createFrom(source: any = {}) {
	        return new CommandContribution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.shortcut = source["shortcut"];
	    }
	}
	export class SettingsSectionContribution {
	    id: string;
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsSectionContribution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	    }
	}
	export class DockBarContribution {
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new DockBarContribution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}
	export class Contributions {
	    editorDockBars?: DockBarContribution[];
	    settingsSections?: SettingsSectionContribution[];
	    commands?: CommandContribution[];
	
	    static createFrom(source: any = {}) {
	        return new Contributions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.editorDockBars = this.convertValues(source["editorDockBars"], DockBarContribution);
	        this.settingsSections = this.convertValues(source["settingsSections"], SettingsSectionContribution);
	        this.commands = this.convertValues(source["commands"], CommandContribution);
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

export namespace readaloud {
	
	export class Status {
	    installed: boolean;
	    dir: string;
	    bytes_total: number;
	    bytes_on_disk: number;
	    missing: string[];
	    native_supported: boolean;
	    native_installed: boolean;
	    native_bytes_total: number;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.dir = source["dir"];
	        this.bytes_total = source["bytes_total"];
	        this.bytes_on_disk = source["bytes_on_disk"];
	        this.missing = source["missing"];
	        this.native_supported = source["native_supported"];
	        this.native_installed = source["native_installed"];
	        this.native_bytes_total = source["native_bytes_total"];
	    }
	}
	export class VerifyResult {
	    installed: boolean;
	    verified: boolean;
	    native_verified: boolean;
	    version: string;
	    installed_at: string;
	    bytes: number;
	    corrupt?: string[];
	    missing?: string[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new VerifyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.verified = source["verified"];
	        this.native_verified = source["native_verified"];
	        this.version = source["version"];
	        this.installed_at = source["installed_at"];
	        this.bytes = source["bytes"];
	        this.corrupt = source["corrupt"];
	        this.missing = source["missing"];
	        this.error = source["error"];
	    }
	}

}

export namespace types {
	
	export class AIProvider {
	    id: string;
	    nickname: string;
	    kind: string;
	    base_url: string;
	    model: string;
	    api_key?: string;
	
	    static createFrom(source: any = {}) {
	        return new AIProvider(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nickname = source["nickname"];
	        this.kind = source["kind"];
	        this.base_url = source["base_url"];
	        this.model = source["model"];
	        this.api_key = source["api_key"];
	    }
	}
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
	export class ContinuityDecision {
	    signal_id: string;
	    status: string;
	    decided_at?: string;
	
	    static createFrom(source: any = {}) {
	        return new ContinuityDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.signal_id = source["signal_id"];
	        this.status = source["status"];
	        this.decided_at = source["decided_at"];
	    }
	}
	export class ContinuityData {
	    decisions?: ContinuityDecision[];
	    version?: number;
	
	    static createFrom(source: any = {}) {
	        return new ContinuityData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.decisions = this.convertValues(source["decisions"], ContinuityDecision);
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
	export class EvidenceKnowledgeState {
	    state: string;
	    character_ids?: string[];
	    character_names?: string[];
	    counterparty_ids?: string[];
	    counterparty_names?: string[];
	    cue: string;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new EvidenceKnowledgeState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.counterparty_ids = source["counterparty_ids"];
	        this.counterparty_names = source["counterparty_names"];
	        this.cue = source["cue"];
	        this.confidence = source["confidence"];
	    }
	}
	export class EvidenceTerm {
	    text: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new EvidenceTerm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.label = source["label"];
	    }
	}
	export class EvidenceRecord {
	    id: string;
	    kind: string;
	    evidence_type: string;
	    chapter_id: string;
	    chapter_index: number;
	    section: string;
	    section_index: number;
	    paragraph_index: number;
	    sentence_index: number;
	    start_offset: number;
	    end_offset: number;
	    text: string;
	    character_ids?: string[];
	    character_names?: string[];
	    named_entities?: EvidenceTerm[];
	    action?: string;
	    time_expressions?: string[];
	    knowledge_states?: EvidenceKnowledgeState[];
	    confidence: number;
	    rationale: string;
	    status: string;
	    source: string;
	    author_text?: string;
	    author_note?: string;
	    pinned?: boolean;
	    reviewed_at?: string;
	
	    static createFrom(source: any = {}) {
	        return new EvidenceRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.evidence_type = source["evidence_type"];
	        this.chapter_id = source["chapter_id"];
	        this.chapter_index = source["chapter_index"];
	        this.section = source["section"];
	        this.section_index = source["section_index"];
	        this.paragraph_index = source["paragraph_index"];
	        this.sentence_index = source["sentence_index"];
	        this.start_offset = source["start_offset"];
	        this.end_offset = source["end_offset"];
	        this.text = source["text"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.named_entities = this.convertValues(source["named_entities"], EvidenceTerm);
	        this.action = source["action"];
	        this.time_expressions = source["time_expressions"];
	        this.knowledge_states = this.convertValues(source["knowledge_states"], EvidenceKnowledgeState);
	        this.confidence = source["confidence"];
	        this.rationale = source["rationale"];
	        this.status = source["status"];
	        this.source = source["source"];
	        this.author_text = source["author_text"];
	        this.author_note = source["author_note"];
	        this.pinned = source["pinned"];
	        this.reviewed_at = source["reviewed_at"];
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
	export class EvidenceData {
	    content_hash: string;
	    chapter_hashes?: Record<string, string>;
	    engine: string;
	    last_analyzed: string;
	    records: EvidenceRecord[];
	    truncated?: boolean;
	    version: number;
	
	    static createFrom(source: any = {}) {
	        return new EvidenceData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content_hash = source["content_hash"];
	        this.chapter_hashes = source["chapter_hashes"];
	        this.engine = source["engine"];
	        this.last_analyzed = source["last_analyzed"];
	        this.records = this.convertValues(source["records"], EvidenceRecord);
	        this.truncated = source["truncated"];
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
	    evidence?: EvidenceData;
	    continuity?: ContinuityData;
	    version?: number;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entity_resolution = this.convertValues(source["entity_resolution"], EntityData);
	        this.relationships = this.convertValues(source["relationships"], RelationshipData);
	        this.story = this.convertValues(source["story"], StoryAnalysisData);
	        this.evidence = this.convertValues(source["evidence"], EvidenceData);
	        this.continuity = this.convertValues(source["continuity"], ContinuityData);
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
	    analysis_enabled: boolean;
	    read_aloud_enabled: boolean;
	    read_aloud_voice: string;
	    read_aloud_speed: number;
	    read_aloud_device: string;
	    read_aloud_threads: string;
	    read_aloud_volume: number;
	    read_aloud_glow: boolean;
	    analysis_cpu_profile: string;
	    characters_lane_view: string;
	    ai_enabled: boolean;
	    ai_mode: string;
	    ai_task_routes?: Record<string, string>;
	    ai_provider: string;
	    ai_providers?: AIProvider[];
	    ai_api_key?: string;
	    has_api_key: boolean;
	    ai_debug_logging: boolean;
	    ai_model: string;
	    ai_local_endpoint?: string;
	    ai_local_model?: string;
	    prose_guide: string;
	    book_font: string;
	    editor_font_size: string;
	    book_font_size: number;
	    book_line_spacing: string;
	    book_drop_caps: boolean;
	    book_trim_size: string;
	    sidebar_panel_width: number;
	    sidebar_active_section: string;
	    update_check_enabled: boolean;
	    plugins_enabled?: Record<string, boolean>;
	    plugin_settings?: Record<string, any>;
	
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
	        this.analysis_enabled = source["analysis_enabled"];
	        this.read_aloud_enabled = source["read_aloud_enabled"];
	        this.read_aloud_voice = source["read_aloud_voice"];
	        this.read_aloud_speed = source["read_aloud_speed"];
	        this.read_aloud_device = source["read_aloud_device"];
	        this.read_aloud_threads = source["read_aloud_threads"];
	        this.read_aloud_volume = source["read_aloud_volume"];
	        this.read_aloud_glow = source["read_aloud_glow"];
	        this.analysis_cpu_profile = source["analysis_cpu_profile"];
	        this.characters_lane_view = source["characters_lane_view"];
	        this.ai_enabled = source["ai_enabled"];
	        this.ai_mode = source["ai_mode"];
	        this.ai_task_routes = source["ai_task_routes"];
	        this.ai_provider = source["ai_provider"];
	        this.ai_providers = this.convertValues(source["ai_providers"], AIProvider);
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
	        this.update_check_enabled = source["update_check_enabled"];
	        this.plugins_enabled = source["plugins_enabled"];
	        this.plugin_settings = source["plugin_settings"];
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
	export class AudioOptions {
	    includeCopyright: boolean;
	    includeFrontMatter: boolean;
	    includeBackMatter: boolean;
	    omitTitlePage: boolean;
	    editionID?: string;
	    formatID?: string;
	    pageSize: string;
	    fontFamily: string;
	    fontSize: number;
	    lineHeight: number;
	    paragraphSpacing: string;
	    slatePage: boolean;
	    numberParagraphs: boolean;
	    pauseBreaks: boolean;
	    pronunciationColumn: boolean;
	    coverPage: boolean;
	    chapterWordCount: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AudioOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeCopyright = source["includeCopyright"];
	        this.includeFrontMatter = source["includeFrontMatter"];
	        this.includeBackMatter = source["includeBackMatter"];
	        this.omitTitlePage = source["omitTitlePage"];
	        this.editionID = source["editionID"];
	        this.formatID = source["formatID"];
	        this.pageSize = source["pageSize"];
	        this.fontFamily = source["fontFamily"];
	        this.fontSize = source["fontSize"];
	        this.lineHeight = source["lineHeight"];
	        this.paragraphSpacing = source["paragraphSpacing"];
	        this.slatePage = source["slatePage"];
	        this.numberParagraphs = source["numberParagraphs"];
	        this.pauseBreaks = source["pauseBreaks"];
	        this.pronunciationColumn = source["pronunciationColumn"];
	        this.coverPage = source["coverPage"];
	        this.chapterWordCount = source["chapterWordCount"];
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
	export class EditionSnapshot {
	    id: string;
	    frozen: string;
	    title?: string;
	    word_count: number;
	    sections: number;
	    members: number;
	    bytes: number;
	
	    static createFrom(source: any = {}) {
	        return new EditionSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.frozen = source["frozen"];
	        this.title = source["title"];
	        this.word_count = source["word_count"];
	        this.sections = source["sections"];
	        this.members = source["members"];
	        this.bytes = source["bytes"];
	    }
	}
	export class EditionWrap {
	    file_name: string;
	    bytes?: number;
	    width?: number;
	    height?: number;
	    size_label?: string;
	    stored: boolean;
	    stored_label?: string;
	    member?: string;
	    source_path?: string;
	    source_checksum?: string;
	    attached?: string;
	    preview_file?: string;
	    preview_width?: number;
	    preview_height?: number;
	
	    static createFrom(source: any = {}) {
	        return new EditionWrap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_name = source["file_name"];
	        this.bytes = source["bytes"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.size_label = source["size_label"];
	        this.stored = source["stored"];
	        this.stored_label = source["stored_label"];
	        this.member = source["member"];
	        this.source_path = source["source_path"];
	        this.source_checksum = source["source_checksum"];
	        this.attached = source["attached"];
	        this.preview_file = source["preview_file"];
	        this.preview_width = source["preview_width"];
	        this.preview_height = source["preview_height"];
	    }
	}
	export class EditionFormat {
	    id: string;
	    kind: string;
	    format?: string;
	    isbn13?: string;
	    registration?: string;
	    edition_statement?: string;
	    publication_date?: string;
	    list_price?: string;
	    status?: string;
	    channels?: string;
	    trim?: string;
	    page_count?: string;
	    paper_stock?: string;
	    binding?: string;
	    bleed?: string;
	    interior?: string;
	    gutter?: string;
	    epub_version?: string;
	    layout?: string;
	    asin?: string;
	    drm?: string;
	    imprint_of_record?: string;
	    territory_rights?: string;
	    rights_notice?: string;
	    lccn?: string;
	    typesetting?: string;
	    export_settings?: Record<string, any>;
	    wrap?: EditionWrap;
	    snapshot_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new EditionFormat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.format = source["format"];
	        this.isbn13 = source["isbn13"];
	        this.registration = source["registration"];
	        this.edition_statement = source["edition_statement"];
	        this.publication_date = source["publication_date"];
	        this.list_price = source["list_price"];
	        this.status = source["status"];
	        this.channels = source["channels"];
	        this.trim = source["trim"];
	        this.page_count = source["page_count"];
	        this.paper_stock = source["paper_stock"];
	        this.binding = source["binding"];
	        this.bleed = source["bleed"];
	        this.interior = source["interior"];
	        this.gutter = source["gutter"];
	        this.epub_version = source["epub_version"];
	        this.layout = source["layout"];
	        this.asin = source["asin"];
	        this.drm = source["drm"];
	        this.imprint_of_record = source["imprint_of_record"];
	        this.territory_rights = source["territory_rights"];
	        this.rights_notice = source["rights_notice"];
	        this.lccn = source["lccn"];
	        this.typesetting = source["typesetting"];
	        this.export_settings = source["export_settings"];
	        this.wrap = this.convertValues(source["wrap"], EditionWrap);
	        this.snapshot_id = source["snapshot_id"];
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
	export class EditionCover {
	    id: string;
	    file: string;
	    thumb_file: string;
	    large_file?: string;
	    width: number;
	    height: number;
	    bytes: number;
	    thumb_width: number;
	    thumb_height: number;
	    thumb_bytes: number;
	    large_width?: number;
	    large_height?: number;
	    large_bytes?: number;
	    encoding: string;
	    quality: number;
	    greyscale?: boolean;
	    converted_from_cmyk?: boolean;
	    flattened_alpha?: boolean;
	    source_path?: string;
	    source_checksum?: string;
	    source_bytes?: number;
	    source_width?: number;
	    source_height?: number;
	    source_modified?: string;
	    source_format?: string;
	    attached?: string;
	    notes?: string[];
	
	    static createFrom(source: any = {}) {
	        return new EditionCover(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.file = source["file"];
	        this.thumb_file = source["thumb_file"];
	        this.large_file = source["large_file"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.bytes = source["bytes"];
	        this.thumb_width = source["thumb_width"];
	        this.thumb_height = source["thumb_height"];
	        this.thumb_bytes = source["thumb_bytes"];
	        this.large_width = source["large_width"];
	        this.large_height = source["large_height"];
	        this.large_bytes = source["large_bytes"];
	        this.encoding = source["encoding"];
	        this.quality = source["quality"];
	        this.greyscale = source["greyscale"];
	        this.converted_from_cmyk = source["converted_from_cmyk"];
	        this.flattened_alpha = source["flattened_alpha"];
	        this.source_path = source["source_path"];
	        this.source_checksum = source["source_checksum"];
	        this.source_bytes = source["source_bytes"];
	        this.source_width = source["source_width"];
	        this.source_height = source["source_height"];
	        this.source_modified = source["source_modified"];
	        this.source_format = source["source_format"];
	        this.attached = source["attached"];
	        this.notes = source["notes"];
	    }
	}
	export class Edition {
	    id: string;
	    label: string;
	    year: string;
	    status: string;
	    cover_id?: string;
	    cover?: EditionCover;
	    previous_edition_id?: string;
	    revision_note?: string;
	    formats: EditionFormat[];
	
	    static createFrom(source: any = {}) {
	        return new Edition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.year = source["year"];
	        this.status = source["status"];
	        this.cover_id = source["cover_id"];
	        this.cover = this.convertValues(source["cover"], EditionCover);
	        this.previous_edition_id = source["previous_edition_id"];
	        this.revision_note = source["revision_note"];
	        this.formats = this.convertValues(source["formats"], EditionFormat);
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
	export class EditionIndex {
	    version: number;
	    editions: Edition[];
	    snapshots?: EditionSnapshot[];
	
	    static createFrom(source: any = {}) {
	        return new EditionIndex(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.editions = this.convertValues(source["editions"], Edition);
	        this.snapshots = this.convertValues(source["snapshots"], EditionSnapshot);
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
	export class PlannerNote {
	    id: string;
	    title: string;
	    body: string;
	    updated?: string;
	    excluded?: boolean;
	    system?: string;
	
	    static createFrom(source: any = {}) {
	        return new PlannerNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.body = source["body"];
	        this.updated = source["updated"];
	        this.excluded = source["excluded"];
	        this.system = source["system"];
	    }
	}
	export class PlannerLink {
	    chapter_id: string;
	    scene: number;
	
	    static createFrom(source: any = {}) {
	        return new PlannerLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter_id = source["chapter_id"];
	        this.scene = source["scene"];
	    }
	}
	export class PlannerCard {
	    id: string;
	    source_id?: string;
	    source_key?: string;
	    origin?: string;
	    title: string;
	    synopsis: string;
	    lines: string[];
	    who: string[];
	    who_names?: string[];
	    changes?: string;
	    stakes?: string;
	    chapter_id: string;
	    link?: PlannerLink;
	    status: string;
	    updated?: string;
	
	    static createFrom(source: any = {}) {
	        return new PlannerCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source_id = source["source_id"];
	        this.source_key = source["source_key"];
	        this.origin = source["origin"];
	        this.title = source["title"];
	        this.synopsis = source["synopsis"];
	        this.lines = source["lines"];
	        this.who = source["who"];
	        this.who_names = source["who_names"];
	        this.changes = source["changes"];
	        this.stakes = source["stakes"];
	        this.chapter_id = source["chapter_id"];
	        this.link = this.convertValues(source["link"], PlannerLink);
	        this.status = source["status"];
	        this.updated = source["updated"];
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
	export class PlannerLane {
	    id: string;
	    name: string;
	    kind: string;
	    color: string;
	    character_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new PlannerLane(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.color = source["color"];
	        this.character_id = source["character_id"];
	    }
	}
	export class PlannerData {
	    version: number;
	    lanes: PlannerLane[];
	    cards: PlannerCard[];
	    notes: PlannerNote[];
	    synopsis?: Record<string, string>;
	    beat_template?: string;
	    hidden_lanes?: string[];
	    compact?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PlannerData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.lanes = this.convertValues(source["lanes"], PlannerLane);
	        this.cards = this.convertValues(source["cards"], PlannerCard);
	        this.notes = this.convertValues(source["notes"], PlannerNote);
	        this.synopsis = source["synopsis"];
	        this.beat_template = source["beat_template"];
	        this.hidden_lanes = source["hidden_lanes"];
	        this.compact = source["compact"];
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
	export class ReadAloudCast {
	    cast_mode: boolean;
	    voices?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new ReadAloudCast(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cast_mode = source["cast_mode"];
	        this.voices = source["voices"];
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
	export class ISBNEntry {
	    format: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new ISBNEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.value = source["value"];
	    }
	}
	export class Metadata {
	    title: string;
	    subtitle?: string;
	    author: string;
	    series_name?: string;
	    series_number?: string;
	    isbn: string;
	    isbns?: ISBNEntry[];
	    publisher: string;
	    imprint?: string;
	    language?: string;
	    copyright_holder?: string;
	    bisac_1?: string;
	    bisac_2?: string;
	    audience?: string;
	    keywords?: string;
	    short_description?: string;
	    contributors?: string;
	    created: string;
	    modified: string;
	    word_count?: number;
	    book_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new Metadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.subtitle = source["subtitle"];
	        this.author = source["author"];
	        this.series_name = source["series_name"];
	        this.series_number = source["series_number"];
	        this.isbn = source["isbn"];
	        this.isbns = this.convertValues(source["isbns"], ISBNEntry);
	        this.publisher = source["publisher"];
	        this.imprint = source["imprint"];
	        this.language = source["language"];
	        this.copyright_holder = source["copyright_holder"];
	        this.bisac_1 = source["bisac_1"];
	        this.bisac_2 = source["bisac_2"];
	        this.audience = source["audience"];
	        this.keywords = source["keywords"];
	        this.short_description = source["short_description"];
	        this.contributors = source["contributors"];
	        this.created = source["created"];
	        this.modified = source["modified"];
	        this.word_count = source["word_count"];
	        this.book_id = source["book_id"];
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
	    read_aloud_cast?: ReadAloudCast;
	    planner?: PlannerData;
	    editions?: EditionIndex;
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
	        this.read_aloud_cast = this.convertValues(source["read_aloud_cast"], ReadAloudCast);
	        this.planner = this.convertValues(source["planner"], PlannerData);
	        this.editions = this.convertValues(source["editions"], EditionIndex);
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
	export class BookLockInfo {
	    held: boolean;
	    stale: boolean;
	    device: string;
	    platform: string;
	    app: string;
	    last_seen: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new BookLockInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.held = source["held"];
	        this.stale = source["stale"];
	        this.device = source["device"];
	        this.platform = source["platform"];
	        this.app = source["app"];
	        this.last_seen = source["last_seen"];
	        this.message = source["message"];
	    }
	}
	export class BookTakeoverRequest {
	    device: string;
	    platform: string;
	    app: string;
	    requested_at: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new BookTakeoverRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device = source["device"];
	        this.platform = source["platform"];
	        this.app = source["app"];
	        this.requested_at = source["requested_at"];
	        this.message = source["message"];
	    }
	}
	export class BookTakeoverResult {
	    granted: boolean;
	    declined: boolean;
	    withdrawn: boolean;
	    fingerprinted: boolean;
	    device?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BookTakeoverResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.granted = source["granted"];
	        this.declined = source["declined"];
	        this.withdrawn = source["withdrawn"];
	        this.fingerprinted = source["fingerprinted"];
	        this.device = source["device"];
	        this.error = source["error"];
	    }
	}
	export class BookTakeoverStatus {
	    asked: boolean;
	    held_elsewhere: boolean;
	    answered: boolean;
	    granted: boolean;
	    declined: boolean;
	    arrived: boolean;
	    unverifiable: boolean;
	    local_bytes: number;
	    expected_bytes: number;
	    responder?: string;
	    note?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new BookTakeoverStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.asked = source["asked"];
	        this.held_elsewhere = source["held_elsewhere"];
	        this.answered = source["answered"];
	        this.granted = source["granted"];
	        this.declined = source["declined"];
	        this.arrived = source["arrived"];
	        this.unverifiable = source["unverifiable"];
	        this.local_bytes = source["local_bytes"];
	        this.expected_bytes = source["expected_bytes"];
	        this.responder = source["responder"];
	        this.note = source["note"];
	        this.message = source["message"];
	    }
	}
	export class PrintPDFOptions {
	    includeCopyright: boolean;
	    includeFrontMatter: boolean;
	    includeBackMatter: boolean;
	    omitTitlePage: boolean;
	    editionID?: string;
	    formatID?: string;
	    pageSize: string;
	    fontFamily: string;
	    fontSize: number;
	    lineHeight: number;
	    paragraphIndent: string;
	    textAlign: string;
	    hideFolios?: boolean;
	    draftWatermark?: boolean;
	    trimSize: string;
	    customWidth: string;
	    customHeight: string;
	    bleed: string;
	    gutterMargin: string;
	    outerMargin: string;
	    topMargin: string;
	    bottomMargin: string;
	    includeCropMarks: boolean;
	    skipKDPChecks?: boolean;
	    chapterStartsRecto: boolean;
	    dropCap: boolean;
	    dropCapLines: number;
	    sceneBreakStyle: string;
	    chapterStyle: string;
	    runningHeaders: boolean;
	    headerStyle: string;
	    headerContent: string;
	    pageNumberPosition: string;
	    generateHalfTitle: boolean;
	    generateTOC: boolean;
	    mirroredMargins: boolean;
	    headingFont: string;
	    furnitureFont: string;
	    titlePageFont: string;
	    titlePageStyle: string;
	    titlePageShowAuthor: boolean;
	    titlePageShowPublisher: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PrintPDFOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeCopyright = source["includeCopyright"];
	        this.includeFrontMatter = source["includeFrontMatter"];
	        this.includeBackMatter = source["includeBackMatter"];
	        this.omitTitlePage = source["omitTitlePage"];
	        this.editionID = source["editionID"];
	        this.formatID = source["formatID"];
	        this.pageSize = source["pageSize"];
	        this.fontFamily = source["fontFamily"];
	        this.fontSize = source["fontSize"];
	        this.lineHeight = source["lineHeight"];
	        this.paragraphIndent = source["paragraphIndent"];
	        this.textAlign = source["textAlign"];
	        this.hideFolios = source["hideFolios"];
	        this.draftWatermark = source["draftWatermark"];
	        this.trimSize = source["trimSize"];
	        this.customWidth = source["customWidth"];
	        this.customHeight = source["customHeight"];
	        this.bleed = source["bleed"];
	        this.gutterMargin = source["gutterMargin"];
	        this.outerMargin = source["outerMargin"];
	        this.topMargin = source["topMargin"];
	        this.bottomMargin = source["bottomMargin"];
	        this.includeCropMarks = source["includeCropMarks"];
	        this.skipKDPChecks = source["skipKDPChecks"];
	        this.chapterStartsRecto = source["chapterStartsRecto"];
	        this.dropCap = source["dropCap"];
	        this.dropCapLines = source["dropCapLines"];
	        this.sceneBreakStyle = source["sceneBreakStyle"];
	        this.chapterStyle = source["chapterStyle"];
	        this.runningHeaders = source["runningHeaders"];
	        this.headerStyle = source["headerStyle"];
	        this.headerContent = source["headerContent"];
	        this.pageNumberPosition = source["pageNumberPosition"];
	        this.generateHalfTitle = source["generateHalfTitle"];
	        this.generateTOC = source["generateTOC"];
	        this.mirroredMargins = source["mirroredMargins"];
	        this.headingFont = source["headingFont"];
	        this.furnitureFont = source["furnitureFont"];
	        this.titlePageFont = source["titlePageFont"];
	        this.titlePageStyle = source["titlePageStyle"];
	        this.titlePageShowAuthor = source["titlePageShowAuthor"];
	        this.titlePageShowPublisher = source["titlePageShowPublisher"];
	    }
	}
	export class DOCXOptions {
	    includeCopyright: boolean;
	    includeFrontMatter: boolean;
	    includeBackMatter: boolean;
	    omitTitlePage: boolean;
	    editionID?: string;
	    formatID?: string;
	    bodyStyle: string;
	    chapterBreak: string;
	    trackChanges: boolean;
	    hashSceneBreaks: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DOCXOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeCopyright = source["includeCopyright"];
	        this.includeFrontMatter = source["includeFrontMatter"];
	        this.includeBackMatter = source["includeBackMatter"];
	        this.omitTitlePage = source["omitTitlePage"];
	        this.editionID = source["editionID"];
	        this.formatID = source["formatID"];
	        this.bodyStyle = source["bodyStyle"];
	        this.chapterBreak = source["chapterBreak"];
	        this.trackChanges = source["trackChanges"];
	        this.hashSceneBreaks = source["hashSceneBreaks"];
	    }
	}
	export class PDFOptions {
	    includeCopyright: boolean;
	    includeFrontMatter: boolean;
	    includeBackMatter: boolean;
	    omitTitlePage: boolean;
	    editionID?: string;
	    formatID?: string;
	    pageSize: string;
	    fontFamily: string;
	    fontSize: number;
	    lineHeight: number;
	    paragraphIndent: string;
	    textAlign: string;
	    hideFolios?: boolean;
	    draftWatermark?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PDFOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeCopyright = source["includeCopyright"];
	        this.includeFrontMatter = source["includeFrontMatter"];
	        this.includeBackMatter = source["includeBackMatter"];
	        this.omitTitlePage = source["omitTitlePage"];
	        this.editionID = source["editionID"];
	        this.formatID = source["formatID"];
	        this.pageSize = source["pageSize"];
	        this.fontFamily = source["fontFamily"];
	        this.fontSize = source["fontSize"];
	        this.lineHeight = source["lineHeight"];
	        this.paragraphIndent = source["paragraphIndent"];
	        this.textAlign = source["textAlign"];
	        this.hideFolios = source["hideFolios"];
	        this.draftWatermark = source["draftWatermark"];
	    }
	}
	export class EPUBOptions {
	    includeCopyright: boolean;
	    includeFrontMatter: boolean;
	    includeBackMatter: boolean;
	    omitTitlePage: boolean;
	    editionID?: string;
	    formatID?: string;
	    fontFamily: string;
	    paragraphStyle: string;
	    textAlign: string;
	    chapterStyle: string;
	    sceneBreakStyle: string;
	    dropCap: boolean;
	    version?: string;
	
	    static createFrom(source: any = {}) {
	        return new EPUBOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeCopyright = source["includeCopyright"];
	        this.includeFrontMatter = source["includeFrontMatter"];
	        this.includeBackMatter = source["includeBackMatter"];
	        this.omitTitlePage = source["omitTitlePage"];
	        this.editionID = source["editionID"];
	        this.formatID = source["formatID"];
	        this.fontFamily = source["fontFamily"];
	        this.paragraphStyle = source["paragraphStyle"];
	        this.textAlign = source["textAlign"];
	        this.chapterStyle = source["chapterStyle"];
	        this.sceneBreakStyle = source["sceneBreakStyle"];
	        this.dropCap = source["dropCap"];
	        this.version = source["version"];
	    }
	}
	export class ExportOptions {
	    includeCopyright: boolean;
	    includeFrontMatter: boolean;
	    includeBackMatter: boolean;
	    omitTitlePage: boolean;
	    editionID?: string;
	    formatID?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.includeCopyright = source["includeCopyright"];
	        this.includeFrontMatter = source["includeFrontMatter"];
	        this.includeBackMatter = source["includeBackMatter"];
	        this.omitTitlePage = source["omitTitlePage"];
	        this.editionID = source["editionID"];
	        this.formatID = source["formatID"];
	    }
	}
	export class BundleItem {
	    format_id: string;
	    output: string;
	    shared: ExportOptions;
	    epub: EPUBOptions;
	    pdf: PDFOptions;
	    audio: AudioOptions;
	    docx: DOCXOptions;
	    print: PrintPDFOptions;
	
	    static createFrom(source: any = {}) {
	        return new BundleItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format_id = source["format_id"];
	        this.output = source["output"];
	        this.shared = this.convertValues(source["shared"], ExportOptions);
	        this.epub = this.convertValues(source["epub"], EPUBOptions);
	        this.pdf = this.convertValues(source["pdf"], PDFOptions);
	        this.audio = this.convertValues(source["audio"], AudioOptions);
	        this.docx = this.convertValues(source["docx"], DOCXOptions);
	        this.print = this.convertValues(source["print"], PrintPDFOptions);
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
	export class BundleRequest {
	    edition_id: string;
	    include_artwork: boolean;
	    items: BundleItem[];
	
	    static createFrom(source: any = {}) {
	        return new BundleRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.edition_id = source["edition_id"];
	        this.include_artwork = source["include_artwork"];
	        this.items = this.convertValues(source["items"], BundleItem);
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
	
	
	export class ContinuityFacet {
	    id: string;
	    label: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ContinuityFacet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.count = source["count"];
	    }
	}
	export class ContinuitySource {
	    evidence_id?: string;
	    text?: string;
	    chapter_id?: string;
	    chapter_index: number;
	    chapter_title: string;
	    section: string;
	    section_index: number;
	    paragraph_index?: number;
	    start_offset?: number;
	
	    static createFrom(source: any = {}) {
	        return new ContinuitySource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.evidence_id = source["evidence_id"];
	        this.text = source["text"];
	        this.chapter_id = source["chapter_id"];
	        this.chapter_index = source["chapter_index"];
	        this.chapter_title = source["chapter_title"];
	        this.section = source["section"];
	        this.section_index = source["section_index"];
	        this.paragraph_index = source["paragraph_index"];
	        this.start_offset = source["start_offset"];
	    }
	}
	export class ContinuitySignal {
	    id: string;
	    kind: string;
	    category: string;
	    severity: string;
	    title: string;
	    detail: string;
	    character_ids?: string[];
	    character_names?: string[];
	    sources?: ContinuitySource[];
	    confidence: number;
	    status?: string;
	
	    static createFrom(source: any = {}) {
	        return new ContinuitySignal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.category = source["category"];
	        this.severity = source["severity"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.sources = this.convertValues(source["sources"], ContinuitySource);
	        this.confidence = source["confidence"];
	        this.status = source["status"];
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
	export class ContinuityReport {
	    success: boolean;
	    error?: string;
	    engine: string;
	    signals: ContinuitySignal[];
	    categories: ContinuityFacet[];
	    characters: ContinuityFacet[];
	    review_count: number;
	    info_count: number;
	    chapters_checked: number;
	
	    static createFrom(source: any = {}) {
	        return new ContinuityReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.engine = source["engine"];
	        this.signals = this.convertValues(source["signals"], ContinuitySignal);
	        this.categories = this.convertValues(source["categories"], ContinuityFacet);
	        this.characters = this.convertValues(source["characters"], ContinuityFacet);
	        this.review_count = source["review_count"];
	        this.info_count = source["info_count"];
	        this.chapters_checked = source["chapters_checked"];
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
	
	
	export class CoverResult {
	    success: boolean;
	    error?: string;
	    cover?: EditionCover;
	    cancelled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CoverResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.cover = this.convertValues(source["cover"], EditionCover);
	        this.cancelled = source["cancelled"];
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
	export class CoverSourceReport {
	    status: string;
	    message: string;
	    path?: string;
	
	    static createFrom(source: any = {}) {
	        return new CoverSourceReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.path = source["path"];
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
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.book = this.convertValues(source["book"], BookData);
	        this.error = source["error"];
	        this.warnings = source["warnings"];
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
	    coverKey: string;
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
	        this.coverKey = source["coverKey"];
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
	
	
	export class RevealResult {
	    success: boolean;
	    error?: string;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new RevealResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.note = source["note"];
	    }
	}
	export class SaveResult {
	    success: boolean;
	    file_path: string;
	    error?: string;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new SaveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.file_path = source["file_path"];
	        this.error = source["error"];
	        this.warnings = source["warnings"];
	    }
	}
	
	
	
	export class SnapshotResult {
	    success: boolean;
	    error?: string;
	    snapshot?: EditionSnapshot;
	    reused?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SnapshotResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.snapshot = this.convertValues(source["snapshot"], EditionSnapshot);
	        this.reused = source["reused"];
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
	
	
	
	
	export class StorySearchChapterSummary {
	    chapter_index: number;
	    chapter_title: string;
	    occurrences: number;
	    evidence_count: number;
	    event_count: number;
	    fact_count: number;
	
	    static createFrom(source: any = {}) {
	        return new StorySearchChapterSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter_index = source["chapter_index"];
	        this.chapter_title = source["chapter_title"];
	        this.occurrences = source["occurrences"];
	        this.evidence_count = source["evidence_count"];
	        this.event_count = source["event_count"];
	        this.fact_count = source["fact_count"];
	    }
	}
	export class StorySearchEntity {
	    id: string;
	    canonical: string;
	    aliases: string[];
	
	    static createFrom(source: any = {}) {
	        return new StorySearchEntity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.canonical = source["canonical"];
	        this.aliases = source["aliases"];
	    }
	}
	export class StorySearchEvidence {
	    id: string;
	    kind: string;
	    evidence_type: string;
	    status: string;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new StorySearchEvidence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.evidence_type = source["evidence_type"];
	        this.status = source["status"];
	        this.confidence = source["confidence"];
	    }
	}
	export class StorySearchSignal {
	    kind: string;
	    title: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new StorySearchSignal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	    }
	}
	export class StorySearchRelatedTerm {
	    text: string;
	    label: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new StorySearchRelatedTerm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.label = source["label"];
	        this.count = source["count"];
	    }
	}
	export class StorySearchKnowledgeState {
	    evidence_id: string;
	    state: string;
	    character_ids?: string[];
	    character_names?: string[];
	    counterparty_ids?: string[];
	    counterparty_names?: string[];
	    chapter_index: number;
	    chapter_title: string;
	    section: string;
	    section_index: number;
	    text: string;
	    cue: string;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new StorySearchKnowledgeState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.evidence_id = source["evidence_id"];
	        this.state = source["state"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.counterparty_ids = source["counterparty_ids"];
	        this.counterparty_names = source["counterparty_names"];
	        this.chapter_index = source["chapter_index"];
	        this.chapter_title = source["chapter_title"];
	        this.section = source["section"];
	        this.section_index = source["section_index"];
	        this.text = source["text"];
	        this.cue = source["cue"];
	        this.confidence = source["confidence"];
	    }
	}
	export class StorySearchInsight {
	    intent: string;
	    interpreted_query: string;
	    chapter_count: number;
	    evidence_count: number;
	    event_count: number;
	    fact_count: number;
	    discovery_count: number;
	    knowledge_count: number;
	    chapters: StorySearchChapterSummary[];
	    knowledge_states?: StorySearchKnowledgeState[];
	    related_terms?: StorySearchRelatedTerm[];
	    signals?: StorySearchSignal[];
	
	    static createFrom(source: any = {}) {
	        return new StorySearchInsight(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.intent = source["intent"];
	        this.interpreted_query = source["interpreted_query"];
	        this.chapter_count = source["chapter_count"];
	        this.evidence_count = source["evidence_count"];
	        this.event_count = source["event_count"];
	        this.fact_count = source["fact_count"];
	        this.discovery_count = source["discovery_count"];
	        this.knowledge_count = source["knowledge_count"];
	        this.chapters = this.convertValues(source["chapters"], StorySearchChapterSummary);
	        this.knowledge_states = this.convertValues(source["knowledge_states"], StorySearchKnowledgeState);
	        this.related_terms = this.convertValues(source["related_terms"], StorySearchRelatedTerm);
	        this.signals = this.convertValues(source["signals"], StorySearchSignal);
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
	
	export class StorySearchMatch {
	    section: string;
	    section_index: number;
	    chapter_index: number;
	    chapter_id?: string;
	    chapter_title: string;
	    scene_index: number;
	    excerpt: string;
	    matched_terms: string[];
	    additional_hits?: number;
	    evidence?: StorySearchEvidence[];
	
	    static createFrom(source: any = {}) {
	        return new StorySearchMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.section = source["section"];
	        this.section_index = source["section_index"];
	        this.chapter_index = source["chapter_index"];
	        this.chapter_id = source["chapter_id"];
	        this.chapter_title = source["chapter_title"];
	        this.scene_index = source["scene_index"];
	        this.excerpt = source["excerpt"];
	        this.matched_terms = source["matched_terms"];
	        this.additional_hits = source["additional_hits"];
	        this.evidence = this.convertValues(source["evidence"], StorySearchEvidence);
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
	
	export class StorySearchRequest {
	    query: string;
	    limit?: number;
	
	    static createFrom(source: any = {}) {
	        return new StorySearchRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.limit = source["limit"];
	    }
	}
	export class StorySearchResult {
	    query: string;
	    matches: StorySearchMatch[];
	    resolved_entities?: StorySearchEntity[];
	    insight?: StorySearchInsight;
	    total: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new StorySearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.matches = this.convertValues(source["matches"], StorySearchMatch);
	        this.resolved_entities = this.convertValues(source["resolved_entities"], StorySearchEntity);
	        this.insight = this.convertValues(source["insight"], StorySearchInsight);
	        this.total = source["total"];
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
	
	export class StoryTimelineChapter {
	    chapter_index: number;
	    chapter_title: string;
	    event_count: number;
	    explicit_time_count: number;
	
	    static createFrom(source: any = {}) {
	        return new StoryTimelineChapter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapter_index = source["chapter_index"];
	        this.chapter_title = source["chapter_title"];
	        this.event_count = source["event_count"];
	        this.explicit_time_count = source["explicit_time_count"];
	    }
	}
	export class StoryTimelineEvent {
	    id: string;
	    evidence_ids: string[];
	    primary_type: string;
	    event_types: string[];
	    text: string;
	    source_text: string;
	    chapter_id: string;
	    chapter_index: number;
	    chapter_title: string;
	    section: string;
	    section_index: number;
	    paragraph_index: number;
	    sentence_index: number;
	    start_offset: number;
	    character_ids?: string[];
	    character_names?: string[];
	    thread_terms?: EvidenceTerm[];
	    locations?: EvidenceTerm[];
	    time_expressions?: string[];
	    time_kind: string;
	    time_label: string;
	    context_id?: string;
	    story_day?: number;
	    narrative_order?: number;
	    importance?: number;
	    confidence: number;
	    status: string;
	    pinned?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StoryTimelineEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.evidence_ids = source["evidence_ids"];
	        this.primary_type = source["primary_type"];
	        this.event_types = source["event_types"];
	        this.text = source["text"];
	        this.source_text = source["source_text"];
	        this.chapter_id = source["chapter_id"];
	        this.chapter_index = source["chapter_index"];
	        this.chapter_title = source["chapter_title"];
	        this.section = source["section"];
	        this.section_index = source["section_index"];
	        this.paragraph_index = source["paragraph_index"];
	        this.sentence_index = source["sentence_index"];
	        this.start_offset = source["start_offset"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.thread_terms = this.convertValues(source["thread_terms"], EvidenceTerm);
	        this.locations = this.convertValues(source["locations"], EvidenceTerm);
	        this.time_expressions = source["time_expressions"];
	        this.time_kind = source["time_kind"];
	        this.time_label = source["time_label"];
	        this.context_id = source["context_id"];
	        this.story_day = source["story_day"];
	        this.narrative_order = source["narrative_order"];
	        this.importance = source["importance"];
	        this.confidence = source["confidence"];
	        this.status = source["status"];
	        this.pinned = source["pinned"];
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
	export class StoryTimelineFacet {
	    id: string;
	    label: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new StoryTimelineFacet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.count = source["count"];
	    }
	}
	export class StoryTimelineResult {
	    success: boolean;
	    error?: string;
	    engine: string;
	    events: StoryTimelineEvent[];
	    chapters: StoryTimelineChapter[];
	    characters: StoryTimelineFacet[];
	    locations: StoryTimelineFacet[];
	    event_types: StoryTimelineFacet[];
	    explicit_time_count: number;
	    relative_time_count: number;
	    chronology_available?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StoryTimelineResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.engine = source["engine"];
	        this.events = this.convertValues(source["events"], StoryTimelineEvent);
	        this.chapters = this.convertValues(source["chapters"], StoryTimelineChapter);
	        this.characters = this.convertValues(source["characters"], StoryTimelineFacet);
	        this.locations = this.convertValues(source["locations"], StoryTimelineFacet);
	        this.event_types = this.convertValues(source["event_types"], StoryTimelineFacet);
	        this.explicit_time_count = source["explicit_time_count"];
	        this.relative_time_count = source["relative_time_count"];
	        this.chronology_available = source["chronology_available"];
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
	
	export class WrapResult {
	    success: boolean;
	    error?: string;
	    wrap?: EditionWrap;
	    cancelled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WrapResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.wrap = this.convertValues(source["wrap"], EditionWrap);
	        this.cancelled = source["cancelled"];
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

