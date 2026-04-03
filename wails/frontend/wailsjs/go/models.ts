export namespace main {
	
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
	    ai_enabled: boolean;
	    ai_mode: string;
	    ai_provider: string;
	    ai_api_key: string;
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
	        this.ai_enabled = source["ai_enabled"];
	        this.ai_mode = source["ai_mode"];
	        this.ai_provider = source["ai_provider"];
	        this.ai_api_key = source["ai_api_key"];
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
	    title: string;
	    subtitle?: string;
	    type: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ChapterItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
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
	    updated_characters: number;
	    characters?: Character[];
	
	    static createFrom(source: any = {}) {
	        return new IndexResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.characters_found = source["characters_found"];
	        this.new_characters = source["new_characters"];
	        this.updated_characters = source["updated_characters"];
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
	
	
	

}

