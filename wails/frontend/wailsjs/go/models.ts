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
	export class StructureDecisionDependency {
	    evidence_id: string;
	    chapter_id: string;
	    paragraph_index: number;
	    content_hash: string;
	
	    static createFrom(source: any = {}) {
	        return new StructureDecisionDependency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.evidence_id = source["evidence_id"];
	        this.chapter_id = source["chapter_id"];
	        this.paragraph_index = source["paragraph_index"];
	        this.content_hash = source["content_hash"];
	    }
	}
	export class StructureAuthorDecision {
	    id: string;
	    target_type: string;
	    target_id?: string;
	    target_signature?: string;
	    action: string;
	    field?: string;
	    value?: string;
	    note?: string;
	    evidence_ids?: string[];
	    dependencies?: StructureDecisionDependency[];
	    changed_dependency_ids?: string[];
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new StructureAuthorDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.target_type = source["target_type"];
	        this.target_id = source["target_id"];
	        this.target_signature = source["target_signature"];
	        this.action = source["action"];
	        this.field = source["field"];
	        this.value = source["value"];
	        this.note = source["note"];
	        this.evidence_ids = source["evidence_ids"];
	        this.dependencies = this.convertValues(source["dependencies"], StructureDecisionDependency);
	        this.changed_dependency_ids = source["changed_dependency_ids"];
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
	export class FingerprintCorrection {
	    id: string;
	    target_id: string;
	    target_signature?: string;
	    kind: string;
	    value: string;
	    evidence_ids?: string[];
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new FingerprintCorrection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.target_id = source["target_id"];
	        this.target_signature = source["target_signature"];
	        this.kind = source["kind"];
	        this.value = source["value"];
	        this.evidence_ids = source["evidence_ids"];
	        this.status = source["status"];
	    }
	}
	export class CanonRule {
	    id: string;
	    subject: string;
	    predicate: string;
	    object: string;
	    polarity: string;
	    context_id?: string;
	    note?: string;
	    evidence_ids?: string[];
	
	    static createFrom(source: any = {}) {
	        return new CanonRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.subject = source["subject"];
	        this.predicate = source["predicate"];
	        this.object = source["object"];
	        this.polarity = source["polarity"];
	        this.context_id = source["context_id"];
	        this.note = source["note"];
	        this.evidence_ids = source["evidence_ids"];
	    }
	}
	export class CheckpointRequirement {
	    text: string;
	    entity_id?: string;
	    predicate?: string;
	    object?: string;
	    satisfied: boolean;
	    evidence_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new CheckpointRequirement(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.entity_id = source["entity_id"];
	        this.predicate = source["predicate"];
	        this.object = source["object"];
	        this.satisfied = source["satisfied"];
	        this.evidence_id = source["evidence_id"];
	    }
	}
	export class StoryCheckpoint {
	    id: string;
	    title: string;
	    description?: string;
	    kind: string;
	    status: string;
	    requirements?: CheckpointRequirement[];
	    negative_conditions?: CheckpointRequirement[];
	    before_chapter?: number;
	    after_chapter?: number;
	    matched_event_ids?: string[];
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new StoryCheckpoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.requirements = this.convertValues(source["requirements"], CheckpointRequirement);
	        this.negative_conditions = this.convertValues(source["negative_conditions"], CheckpointRequirement);
	        this.before_chapter = source["before_chapter"];
	        this.after_chapter = source["after_chapter"];
	        this.matched_event_ids = source["matched_event_ids"];
	        this.source = source["source"];
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
	export class StoryAuthorModel {
	    contexts?: StoryContext[];
	    checkpoints?: StoryCheckpoint[];
	    canon?: CanonRule[];
	    corrections?: FingerprintCorrection[];
	    structure_decisions?: StructureAuthorDecision[];
	    profiles?: string[];
	    voice_notes?: CharacterVoiceNotes[];
	
	    static createFrom(source: any = {}) {
	        return new StoryAuthorModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.contexts = this.convertValues(source["contexts"], StoryContext);
	        this.checkpoints = this.convertValues(source["checkpoints"], StoryCheckpoint);
	        this.canon = this.convertValues(source["canon"], CanonRule);
	        this.corrections = this.convertValues(source["corrections"], FingerprintCorrection);
	        this.structure_decisions = this.convertValues(source["structure_decisions"], StructureAuthorDecision);
	        this.profiles = source["profiles"];
	        this.voice_notes = this.convertValues(source["voice_notes"], CharacterVoiceNotes);
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
	export class StoryStructureBlockCache {
	    id: string;
	    chapter_id: string;
	    content_hash: string;
	    chapter_index: number;
	    paragraph_index: number;
	    evidence_ids?: string[];
	    aggregate_ids?: string[];
	
	    static createFrom(source: any = {}) {
	        return new StoryStructureBlockCache(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.chapter_id = source["chapter_id"];
	        this.content_hash = source["content_hash"];
	        this.chapter_index = source["chapter_index"];
	        this.paragraph_index = source["paragraph_index"];
	        this.evidence_ids = source["evidence_ids"];
	        this.aggregate_ids = source["aggregate_ids"];
	    }
	}
	export class StoryStructureCache {
	    source_blocks: StoryStructureBlockCache[];
	    evidence_dependents: Record<string, Array<string>>;
	    aggregate_parents: Record<string, Array<string>>;
	
	    static createFrom(source: any = {}) {
	        return new StoryStructureCache(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_blocks = this.convertValues(source["source_blocks"], StoryStructureBlockCache);
	        this.evidence_dependents = source["evidence_dependents"];
	        this.aggregate_parents = source["aggregate_parents"];
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
	export class StoryArc {
	    id: string;
	    label: string;
	    thread_ids: string[];
	    sequence_ids: string[];
	    scene_ids: string[];
	    event_ids: string[];
	    evidence_ids: string[];
	    narrative_start: number;
	    narrative_end: number;
	    salience: number;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new StoryArc(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.thread_ids = source["thread_ids"];
	        this.sequence_ids = source["sequence_ids"];
	        this.scene_ids = source["scene_ids"];
	        this.event_ids = source["event_ids"];
	        this.evidence_ids = source["evidence_ids"];
	        this.narrative_start = source["narrative_start"];
	        this.narrative_end = source["narrative_end"];
	        this.salience = source["salience"];
	        this.confidence = source["confidence"];
	    }
	}
	export class NarrativeThread {
	    id: string;
	    label: string;
	    state: string;
	    sequence_ids: string[];
	    scene_ids: string[];
	    event_ids: string[];
	    evidence_ids: string[];
	    character_ids?: string[];
	    character_names?: string[];
	    objective_terms?: string[];
	    location_terms?: string[];
	    context_ids?: string[];
	    obligation_thread_ids?: string[];
	    parent_ids?: string[];
	    child_ids?: string[];
	    convergence_event_ids?: string[];
	    separation_event_ids?: string[];
	    narrative_start: number;
	    narrative_end: number;
	    salience: number;
	    confidence: number;
	    creation_decision: StructureDecision;
	    continuation_decisions?: StructureDecision[];
	    convergence_decisions?: StructureDecision[];
	    separation_decisions?: StructureDecision[];
	
	    static createFrom(source: any = {}) {
	        return new NarrativeThread(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.state = source["state"];
	        this.sequence_ids = source["sequence_ids"];
	        this.scene_ids = source["scene_ids"];
	        this.event_ids = source["event_ids"];
	        this.evidence_ids = source["evidence_ids"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.objective_terms = source["objective_terms"];
	        this.location_terms = source["location_terms"];
	        this.context_ids = source["context_ids"];
	        this.obligation_thread_ids = source["obligation_thread_ids"];
	        this.parent_ids = source["parent_ids"];
	        this.child_ids = source["child_ids"];
	        this.convergence_event_ids = source["convergence_event_ids"];
	        this.separation_event_ids = source["separation_event_ids"];
	        this.narrative_start = source["narrative_start"];
	        this.narrative_end = source["narrative_end"];
	        this.salience = source["salience"];
	        this.confidence = source["confidence"];
	        this.creation_decision = this.convertValues(source["creation_decision"], StructureDecision);
	        this.continuation_decisions = this.convertValues(source["continuation_decisions"], StructureDecision);
	        this.convergence_decisions = this.convertValues(source["convergence_decisions"], StructureDecision);
	        this.separation_decisions = this.convertValues(source["separation_decisions"], StructureDecision);
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
	export class StorySequence {
	    id: string;
	    summary: string;
	    scene_ids: string[];
	    event_ids: string[];
	    evidence_ids: string[];
	    character_ids?: string[];
	    character_names?: string[];
	    context_ids?: string[];
	    locations?: EvidenceTerm[];
	    objective_terms?: EvidenceTerm[];
	    narrative_start: number;
	    narrative_end: number;
	    salience: number;
	    confidence: number;
	    creation_decision: StructureDecision;
	    membership_decisions?: StructureDecision[];
	    boundary_decision: StructureDecision;
	
	    static createFrom(source: any = {}) {
	        return new StorySequence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.summary = source["summary"];
	        this.scene_ids = source["scene_ids"];
	        this.event_ids = source["event_ids"];
	        this.evidence_ids = source["evidence_ids"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.context_ids = source["context_ids"];
	        this.locations = this.convertValues(source["locations"], EvidenceTerm);
	        this.objective_terms = this.convertValues(source["objective_terms"], EvidenceTerm);
	        this.narrative_start = source["narrative_start"];
	        this.narrative_end = source["narrative_end"];
	        this.salience = source["salience"];
	        this.confidence = source["confidence"];
	        this.creation_decision = this.convertValues(source["creation_decision"], StructureDecision);
	        this.membership_decisions = this.convertValues(source["membership_decisions"], StructureDecision);
	        this.boundary_decision = this.convertValues(source["boundary_decision"], StructureDecision);
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
	export class SemanticScene {
	    id: string;
	    summary: string;
	    event_ids: string[];
	    evidence_ids: string[];
	    character_ids?: string[];
	    character_names?: string[];
	    locations?: EvidenceTerm[];
	    objective_terms?: EvidenceTerm[];
	    context_id: string;
	    story_time: StoryTime;
	    chapter_ids: string[];
	    chapter_start: number;
	    chapter_end: number;
	    narrative_start: number;
	    narrative_end: number;
	    start_offset: number;
	    end_offset: number;
	    paragraph_end: number;
	    salience: number;
	    confidence: number;
	    boundary_confidence: number;
	    boundary_source: string;
	    creation_decision: StructureDecision;
	    membership_decisions?: StructureDecision[];
	    boundary_decision: StructureDecision;
	    temporal_decision: StructureDecision;
	
	    static createFrom(source: any = {}) {
	        return new SemanticScene(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.summary = source["summary"];
	        this.event_ids = source["event_ids"];
	        this.evidence_ids = source["evidence_ids"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.locations = this.convertValues(source["locations"], EvidenceTerm);
	        this.objective_terms = this.convertValues(source["objective_terms"], EvidenceTerm);
	        this.context_id = source["context_id"];
	        this.story_time = this.convertValues(source["story_time"], StoryTime);
	        this.chapter_ids = source["chapter_ids"];
	        this.chapter_start = source["chapter_start"];
	        this.chapter_end = source["chapter_end"];
	        this.narrative_start = source["narrative_start"];
	        this.narrative_end = source["narrative_end"];
	        this.start_offset = source["start_offset"];
	        this.end_offset = source["end_offset"];
	        this.paragraph_end = source["paragraph_end"];
	        this.salience = source["salience"];
	        this.confidence = source["confidence"];
	        this.boundary_confidence = source["boundary_confidence"];
	        this.boundary_source = source["boundary_source"];
	        this.creation_decision = this.convertValues(source["creation_decision"], StructureDecision);
	        this.membership_decisions = this.convertValues(source["membership_decisions"], StructureDecision);
	        this.boundary_decision = this.convertValues(source["boundary_decision"], StructureDecision);
	        this.temporal_decision = this.convertValues(source["temporal_decision"], StructureDecision);
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
	export class StructureSignal {
	    code: string;
	    detail: string;
	    weight?: number;
	    evidence_ids?: string[];
	
	    static createFrom(source: any = {}) {
	        return new StructureSignal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.detail = source["detail"];
	        this.weight = source["weight"];
	        this.evidence_ids = source["evidence_ids"];
	    }
	}
	export class StructureDecision {
	    kind: string;
	    outcome: string;
	    rule: string;
	    summary: string;
	    child_id?: string;
	    candidate_id?: string;
	    score?: number;
	    threshold?: number;
	    confidence: number;
	    signals?: StructureSignal[];
	
	    static createFrom(source: any = {}) {
	        return new StructureDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.outcome = source["outcome"];
	        this.rule = source["rule"];
	        this.summary = source["summary"];
	        this.child_id = source["child_id"];
	        this.candidate_id = source["candidate_id"];
	        this.score = source["score"];
	        this.threshold = source["threshold"];
	        this.confidence = source["confidence"];
	        this.signals = this.convertValues(source["signals"], StructureSignal);
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
	export class StructureEvidenceRef {
	    evidence_id: string;
	    chapter_id: string;
	    chapter_index: number;
	    section: string;
	    section_index: number;
	    paragraph_index: number;
	    sentence_index: number;
	    start_offset: number;
	    end_offset: number;
	    quotation: string;
	    confidence: number;
	    status: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new StructureEvidenceRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.evidence_id = source["evidence_id"];
	        this.chapter_id = source["chapter_id"];
	        this.chapter_index = source["chapter_index"];
	        this.section = source["section"];
	        this.section_index = source["section_index"];
	        this.paragraph_index = source["paragraph_index"];
	        this.sentence_index = source["sentence_index"];
	        this.start_offset = source["start_offset"];
	        this.end_offset = source["end_offset"];
	        this.quotation = source["quotation"];
	        this.confidence = source["confidence"];
	        this.status = source["status"];
	        this.source = source["source"];
	    }
	}
	export class SalienceBreakdown {
	    base: number;
	    state_change?: number;
	    movement?: number;
	    decision?: number;
	    discovery?: number;
	    temporal_change?: number;
	    thread_change?: number;
	    later_references?: number;
	    contradiction?: number;
	    description_only?: number;
	    low_confidence?: number;
	
	    static createFrom(source: any = {}) {
	        return new SalienceBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base = source["base"];
	        this.state_change = source["state_change"];
	        this.movement = source["movement"];
	        this.decision = source["decision"];
	        this.discovery = source["discovery"];
	        this.temporal_change = source["temporal_change"];
	        this.thread_change = source["thread_change"];
	        this.later_references = source["later_references"];
	        this.contradiction = source["contradiction"];
	        this.description_only = source["description_only"];
	        this.low_confidence = source["low_confidence"];
	    }
	}
	export class SignificantStoryEvent {
	    id: string;
	    summary: string;
	    author_summary?: string;
	    evidence_ids: string[];
	    fingerprint_event_ids: string[];
	    assertion_ids?: string[];
	    character_ids?: string[];
	    character_names?: string[];
	    kinds?: string[];
	    locations?: EvidenceTerm[];
	    objects?: EvidenceTerm[];
	    state_ids?: string[];
	    obligation_thread_ids?: string[];
	    context_id: string;
	    story_time: StoryTime;
	    chapter_id: string;
	    chapter_index: number;
	    chapter_title: string;
	    section: string;
	    section_index: number;
	    paragraph_start: number;
	    paragraph_end: number;
	    start_offset: number;
	    end_offset: number;
	    narrative_order: number;
	    salience: number;
	    salience_reasons: SalienceBreakdown;
	    evidence_refs: StructureEvidenceRef[];
	    creation_decision: StructureDecision;
	    membership_decisions?: StructureDecision[];
	    boundary_decisions?: StructureDecision[];
	    salience_signals?: StructureSignal[];
	    temporal_decision: StructureDecision;
	    author_decision_ids?: string[];
	    correction_ids?: string[];
	    interpretation_status: string;
	    interpretation_signals?: StructureSignal[];
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new SignificantStoryEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.summary = source["summary"];
	        this.author_summary = source["author_summary"];
	        this.evidence_ids = source["evidence_ids"];
	        this.fingerprint_event_ids = source["fingerprint_event_ids"];
	        this.assertion_ids = source["assertion_ids"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.kinds = source["kinds"];
	        this.locations = this.convertValues(source["locations"], EvidenceTerm);
	        this.objects = this.convertValues(source["objects"], EvidenceTerm);
	        this.state_ids = source["state_ids"];
	        this.obligation_thread_ids = source["obligation_thread_ids"];
	        this.context_id = source["context_id"];
	        this.story_time = this.convertValues(source["story_time"], StoryTime);
	        this.chapter_id = source["chapter_id"];
	        this.chapter_index = source["chapter_index"];
	        this.chapter_title = source["chapter_title"];
	        this.section = source["section"];
	        this.section_index = source["section_index"];
	        this.paragraph_start = source["paragraph_start"];
	        this.paragraph_end = source["paragraph_end"];
	        this.start_offset = source["start_offset"];
	        this.end_offset = source["end_offset"];
	        this.narrative_order = source["narrative_order"];
	        this.salience = source["salience"];
	        this.salience_reasons = this.convertValues(source["salience_reasons"], SalienceBreakdown);
	        this.evidence_refs = this.convertValues(source["evidence_refs"], StructureEvidenceRef);
	        this.creation_decision = this.convertValues(source["creation_decision"], StructureDecision);
	        this.membership_decisions = this.convertValues(source["membership_decisions"], StructureDecision);
	        this.boundary_decisions = this.convertValues(source["boundary_decisions"], StructureDecision);
	        this.salience_signals = this.convertValues(source["salience_signals"], StructureSignal);
	        this.temporal_decision = this.convertValues(source["temporal_decision"], StructureDecision);
	        this.author_decision_ids = source["author_decision_ids"];
	        this.correction_ids = source["correction_ids"];
	        this.interpretation_status = source["interpretation_status"];
	        this.interpretation_signals = this.convertValues(source["interpretation_signals"], StructureSignal);
	        this.confidence = source["confidence"];
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
	export class StoryStructure {
	    content_hash: string;
	    engine: string;
	    last_analyzed: string;
	    version: number;
	    significant_events: SignificantStoryEvent[];
	    scenes: SemanticScene[];
	    sequences: StorySequence[];
	    narrative_threads: NarrativeThread[];
	    arcs: StoryArc[];
	    cache: StoryStructureCache;
	
	    static createFrom(source: any = {}) {
	        return new StoryStructure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content_hash = source["content_hash"];
	        this.engine = source["engine"];
	        this.last_analyzed = source["last_analyzed"];
	        this.version = source["version"];
	        this.significant_events = this.convertValues(source["significant_events"], SignificantStoryEvent);
	        this.scenes = this.convertValues(source["scenes"], SemanticScene);
	        this.sequences = this.convertValues(source["sequences"], StorySequence);
	        this.narrative_threads = this.convertValues(source["narrative_threads"], NarrativeThread);
	        this.arcs = this.convertValues(source["arcs"], StoryArc);
	        this.cache = this.convertValues(source["cache"], StoryStructureCache);
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
	export class CharacterVoiceNotes {
	    character_id: string;
	    dialect?: string;
	    vernacular?: string[];
	    speaking_traits?: string[];
	    avoids?: string[];
	    notes?: string;
	
	    static createFrom(source: any = {}) {
	        return new CharacterVoiceNotes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.character_id = source["character_id"];
	        this.dialect = source["dialect"];
	        this.vernacular = source["vernacular"];
	        this.speaking_traits = source["speaking_traits"];
	        this.avoids = source["avoids"];
	        this.notes = source["notes"];
	    }
	}
	export class DialogueSample {
	    id: string;
	    text: string;
	    chapter_id: string;
	    chapter_index: number;
	    chapter_title: string;
	    start_offset: number;
	    attribution_cue: string;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new DialogueSample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.chapter_id = source["chapter_id"];
	        this.chapter_index = source["chapter_index"];
	        this.chapter_title = source["chapter_title"];
	        this.start_offset = source["start_offset"];
	        this.attribution_cue = source["attribution_cue"];
	        this.confidence = source["confidence"];
	    }
	}
	export class VoiceSignal {
	    kind: string;
	    label: string;
	    count: number;
	    examples?: string[];
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new VoiceSignal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.count = source["count"];
	        this.examples = source["examples"];
	        this.confidence = source["confidence"];
	    }
	}
	export class VoiceTerm {
	    text: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new VoiceTerm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.count = source["count"];
	    }
	}
	export class CharacterVoiceProfile {
	    character_id: string;
	    character_name: string;
	    sample_count: number;
	    word_count: number;
	    average_words: number;
	    contraction_percent: number;
	    question_percent: number;
	    exclamation_percent: number;
	    vocabulary?: VoiceTerm[];
	    address_forms?: VoiceTerm[];
	    dialect_signals?: VoiceSignal[];
	    samples?: DialogueSample[];
	    confidence: number;
	    author_notes?: CharacterVoiceNotes;
	
	    static createFrom(source: any = {}) {
	        return new CharacterVoiceProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.character_id = source["character_id"];
	        this.character_name = source["character_name"];
	        this.sample_count = source["sample_count"];
	        this.word_count = source["word_count"];
	        this.average_words = source["average_words"];
	        this.contraction_percent = source["contraction_percent"];
	        this.question_percent = source["question_percent"];
	        this.exclamation_percent = source["exclamation_percent"];
	        this.vocabulary = this.convertValues(source["vocabulary"], VoiceTerm);
	        this.address_forms = this.convertValues(source["address_forms"], VoiceTerm);
	        this.dialect_signals = this.convertValues(source["dialect_signals"], VoiceSignal);
	        this.samples = this.convertValues(source["samples"], DialogueSample);
	        this.confidence = source["confidence"];
	        this.author_notes = this.convertValues(source["author_notes"], CharacterVoiceNotes);
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
	export class StoryProfile {
	    id: string;
	    label: string;
	    confidence: number;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new StoryProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.confidence = source["confidence"];
	        this.source = source["source"];
	    }
	}
	export class FingerprintDiagnostic {
	    id: string;
	    kind: string;
	    severity: string;
	    title: string;
	    detail: string;
	    event_ids?: string[];
	    evidence_ids?: string[];
	    chapter_indices?: number[];
	    confidence: number;
	    status?: string;
	
	    static createFrom(source: any = {}) {
	        return new FingerprintDiagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.severity = source["severity"];
	        this.title = source["title"];
	        this.detail = source["detail"];
	        this.event_ids = source["event_ids"];
	        this.evidence_ids = source["evidence_ids"];
	        this.chapter_indices = source["chapter_indices"];
	        this.confidence = source["confidence"];
	        this.status = source["status"];
	    }
	}
	export class StoryThread {
	    id: string;
	    label: string;
	    kind: string;
	    state: string;
	    opened_by_event_id?: string;
	    resolved_by_event_id?: string;
	    event_ids?: string[];
	    evidence_ids?: string[];
	    entity_ids?: string[];
	    parent_ids?: string[];
	    child_ids?: string[];
	    resolution: number;
	    dormant_chapters?: number;
	    dormant_words?: number;
	    confidence: number;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new StoryThread(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.kind = source["kind"];
	        this.state = source["state"];
	        this.opened_by_event_id = source["opened_by_event_id"];
	        this.resolved_by_event_id = source["resolved_by_event_id"];
	        this.event_ids = source["event_ids"];
	        this.evidence_ids = source["evidence_ids"];
	        this.entity_ids = source["entity_ids"];
	        this.parent_ids = source["parent_ids"];
	        this.child_ids = source["child_ids"];
	        this.resolution = source["resolution"];
	        this.dormant_chapters = source["dormant_chapters"];
	        this.dormant_words = source["dormant_words"];
	        this.confidence = source["confidence"];
	        this.source = source["source"];
	    }
	}
	export class StoryStateInterval {
	    id: string;
	    entity_id: string;
	    entity_name: string;
	    kind: string;
	    value: string;
	    qualifier?: string;
	    context_id: string;
	    start_event_id: string;
	    end_event_id?: string;
	    evidence_ids: string[];
	    persistent: boolean;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new StoryStateInterval(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.entity_id = source["entity_id"];
	        this.entity_name = source["entity_name"];
	        this.kind = source["kind"];
	        this.value = source["value"];
	        this.qualifier = source["qualifier"];
	        this.context_id = source["context_id"];
	        this.start_event_id = source["start_event_id"];
	        this.end_event_id = source["end_event_id"];
	        this.evidence_ids = source["evidence_ids"];
	        this.persistent = source["persistent"];
	        this.confidence = source["confidence"];
	    }
	}
	export class FingerprintEvent {
	    id: string;
	    summary: string;
	    author_summary?: string;
	    evidence_ids: string[];
	    assertion_ids?: string[];
	    character_ids?: string[];
	    character_names?: string[];
	    locations?: EvidenceTerm[];
	    objects?: EvidenceTerm[];
	    kinds: string[];
	    context_id: string;
	    story_time: StoryTime;
	    chapter_id: string;
	    chapter_index: number;
	    chapter_title: string;
	    paragraph_index: number;
	    start_offset: number;
	    narrative_order: number;
	    importance: number;
	    confidence: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new FingerprintEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.summary = source["summary"];
	        this.author_summary = source["author_summary"];
	        this.evidence_ids = source["evidence_ids"];
	        this.assertion_ids = source["assertion_ids"];
	        this.character_ids = source["character_ids"];
	        this.character_names = source["character_names"];
	        this.locations = this.convertValues(source["locations"], EvidenceTerm);
	        this.objects = this.convertValues(source["objects"], EvidenceTerm);
	        this.kinds = source["kinds"];
	        this.context_id = source["context_id"];
	        this.story_time = this.convertValues(source["story_time"], StoryTime);
	        this.chapter_id = source["chapter_id"];
	        this.chapter_index = source["chapter_index"];
	        this.chapter_title = source["chapter_title"];
	        this.paragraph_index = source["paragraph_index"];
	        this.start_offset = source["start_offset"];
	        this.narrative_order = source["narrative_order"];
	        this.importance = source["importance"];
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
	export class NarrativePromotionStats {
	    evidence_atoms: number;
	    assertions: number;
	    promoted_fingerprints: number;
	    retained_as_evidence: number;

	    static createFrom(source: any = {}) {
	        return new NarrativePromotionStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.evidence_atoms = source["evidence_atoms"];
	        this.assertions = source["assertions"];
	        this.promoted_fingerprints = source["promoted_fingerprints"];
	        this.retained_as_evidence = source["retained_as_evidence"];
	    }
	}
	export class NarrativeFingerprintRelation {
	    id: string;
	    from_id: string;
	    to_id: string;
	    kind: string;
	    explanation: string;
	    evidence_ids?: string[];
	    confidence: number;

	    static createFrom(source: any = {}) {
	        return new NarrativeFingerprintRelation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.from_id = source["from_id"];
	        this.to_id = source["to_id"];
	        this.kind = source["kind"];
	        this.explanation = source["explanation"];
	        this.evidence_ids = source["evidence_ids"];
	        this.confidence = source["confidence"];
	    }
	}
	export class NarrativePromotionReason {
	    code: string;
	    explanation: string;
	    assertion_ids?: string[];
	    evidence_ids?: string[];
	    confidence: number;

	    static createFrom(source: any = {}) {
	        return new NarrativePromotionReason(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.explanation = source["explanation"];
	        this.assertion_ids = source["assertion_ids"];
	        this.evidence_ids = source["evidence_ids"];
	        this.confidence = source["confidence"];
	    }
	}
	export class NarrativeParticipant {
	    entity_id?: string;
	    entity_name: string;
	    role: string;

	    static createFrom(source: any = {}) {
	        return new NarrativeParticipant(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entity_id = source["entity_id"];
	        this.entity_name = source["entity_name"];
	        this.role = source["role"];
	    }
	}
	export class NarrativeEvidenceSpan {
	    evidence_id: string;
	    chapter_id: string;
	    chapter_index: number;
	    section: string;
	    section_index: number;
	    paragraph_index: number;
	    sentence_index: number;
	    start_offset: number;
	    end_offset: number;
	    quote: string;
	    confidence: number;

	    static createFrom(source: any = {}) {
	        return new NarrativeEvidenceSpan(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.evidence_id = source["evidence_id"];
	        this.chapter_id = source["chapter_id"];
	        this.chapter_index = source["chapter_index"];
	        this.section = source["section"];
	        this.section_index = source["section_index"];
	        this.paragraph_index = source["paragraph_index"];
	        this.sentence_index = source["sentence_index"];
	        this.start_offset = source["start_offset"];
	        this.end_offset = source["end_offset"];
	        this.quote = source["quote"];
	        this.confidence = source["confidence"];
	    }
	}
	export class NarrativeFingerprint {
	    id: string;
	    kind: string;
	    summary: string;
	    semantic_key: string;
	    assertion_ids: string[];
	    evidence_ids: string[];
	    evidence_spans: NarrativeEvidenceSpan[];
	    participants?: NarrativeParticipant[];
	    state_change?: NarrativeStateChange;
	    epistemic_status: string;
	    attribution: NarrativeAttribution;
	    scope: NarrativeRealityScope;
	    persistence: string;
	    temporal: StoryTime;
	    promotion_reasons: NarrativePromotionReason[];
	    confidence: number;
	    status: string;

	    static createFrom(source: any = {}) {
	        return new NarrativeFingerprint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.summary = source["summary"];
	        this.semantic_key = source["semantic_key"];
	        this.assertion_ids = source["assertion_ids"];
	        this.evidence_ids = source["evidence_ids"];
	        this.evidence_spans = this.convertValues(source["evidence_spans"], NarrativeEvidenceSpan);
	        this.participants = this.convertValues(source["participants"], NarrativeParticipant);
	        this.state_change = this.convertValues(source["state_change"], NarrativeStateChange);
	        this.epistemic_status = source["epistemic_status"];
	        this.attribution = this.convertValues(source["attribution"], NarrativeAttribution);
	        this.scope = this.convertValues(source["scope"], NarrativeRealityScope);
	        this.persistence = source["persistence"];
	        this.temporal = this.convertValues(source["temporal"], StoryTime);
	        this.promotion_reasons = this.convertValues(source["promotion_reasons"], NarrativePromotionReason);
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
	export class StoryTime {
	    context_id: string;
	    label?: string;
	    day_offset?: number;
	    earliest_day?: number;
	    latest_day?: number;
	    precision: string;
	    confidence: number;

	    static createFrom(source: any = {}) {
	        return new StoryTime(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.context_id = source["context_id"];
	        this.label = source["label"];
	        this.day_offset = source["day_offset"];
	        this.earliest_day = source["earliest_day"];
	        this.latest_day = source["latest_day"];
	        this.precision = source["precision"];
	        this.confidence = source["confidence"];
	    }
	}
	export class NarrativeStateChange {
	    entity_id?: string;
	    entity_name?: string;
	    state_kind: string;
	    previous?: string;
	    new: string;
	    operation: string;

	    static createFrom(source: any = {}) {
	        return new NarrativeStateChange(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entity_id = source["entity_id"];
	        this.entity_name = source["entity_name"];
	        this.state_kind = source["state_kind"];
	        this.previous = source["previous"];
	        this.new = source["new"];
	        this.operation = source["operation"];
	    }
	}
	export class NarrativeRealityScope {
	    id: string;
	    kind: string;
	    label: string;
	    parent_id?: string;
	    evidence_ids?: string[];
	    confidence: number;

	    static createFrom(source: any = {}) {
	        return new NarrativeRealityScope(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.parent_id = source["parent_id"];
	        this.evidence_ids = source["evidence_ids"];
	        this.confidence = source["confidence"];
	    }
	}
	export class NarrativeAttribution {
	    kind: string;
	    entity_id?: string;
	    entity_name?: string;
	    cue?: string;
	    confidence: number;

	    static createFrom(source: any = {}) {
	        return new NarrativeAttribution(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.entity_id = source["entity_id"];
	        this.entity_name = source["entity_name"];
	        this.cue = source["cue"];
	        this.confidence = source["confidence"];
	    }
	}
	export class StoryAssertion {
	    id: string;
	    evidence_ids: string[];
	    kind: string;
	    statement: string;
	    semantic_key: string;
	    subject_id?: string;
	    subject?: string;
	    predicate: string;
	    object_id?: string;
	    object?: string;
	    posture: string;
	    polarity: string;
	    epistemic_status: string;
	    attribution: NarrativeAttribution;
	    scope: NarrativeRealityScope;
	    state_change?: NarrativeStateChange;
	    persistence: string;
	    temporal: StoryTime;
	    status: string;
	    context_id: string;
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new StoryAssertion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.evidence_ids = source["evidence_ids"];
	        this.kind = source["kind"];
	        this.statement = source["statement"];
	        this.semantic_key = source["semantic_key"];
	        this.subject_id = source["subject_id"];
	        this.subject = source["subject"];
	        this.predicate = source["predicate"];
	        this.object_id = source["object_id"];
	        this.object = source["object"];
	        this.posture = source["posture"];
	        this.polarity = source["polarity"];
	        this.epistemic_status = source["epistemic_status"];
	        this.attribution = this.convertValues(source["attribution"], NarrativeAttribution);
	        this.scope = this.convertValues(source["scope"], NarrativeRealityScope);
	        this.state_change = this.convertValues(source["state_change"], NarrativeStateChange);
	        this.persistence = source["persistence"];
	        this.temporal = this.convertValues(source["temporal"], StoryTime);
	        this.status = source["status"];
	        this.context_id = source["context_id"];
	        this.confidence = source["confidence"];
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
	export class TemporalConstraint {
	    id: string;
	    from_evidence_id: string;
	    to_evidence_id?: string;
	    relation: string;
	    offset_days?: number;
	    label: string;
	    posture: string;
	    confidence: number;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new TemporalConstraint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.from_evidence_id = source["from_evidence_id"];
	        this.to_evidence_id = source["to_evidence_id"];
	        this.relation = source["relation"];
	        this.offset_days = source["offset_days"];
	        this.label = source["label"];
	        this.posture = source["posture"];
	        this.confidence = source["confidence"];
	        this.source = source["source"];
	    }
	}
	export class StoryContext {
	    id: string;
	    kind: string;
	    label: string;
	    parent_id?: string;
	    evidence_ids?: string[];
	    confidence: number;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new StoryContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.parent_id = source["parent_id"];
	        this.evidence_ids = source["evidence_ids"];
	        this.confidence = source["confidence"];
	        this.source = source["source"];
	    }
	}
	export class StoryFingerprint {
	    content_hash: string;
	    engine: string;
	    last_analyzed: string;
	    version: number;
	    contexts: StoryContext[];
	    temporal_constraints: TemporalConstraint[];
	    assertions: StoryAssertion[];
	    narrative_fingerprints: NarrativeFingerprint[];
	    narrative_relations?: NarrativeFingerprintRelation[];
	    promotion_stats: NarrativePromotionStats;
	    events: FingerprintEvent[];
	    states: StoryStateInterval[];
	    threads: StoryThread[];
	    diagnostics: FingerprintDiagnostic[];
	    profiles?: StoryProfile[];
	    voices?: CharacterVoiceProfile[];
	    structure?: StoryStructure;
	    author_model: StoryAuthorModel;
	
	    static createFrom(source: any = {}) {
	        return new StoryFingerprint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content_hash = source["content_hash"];
	        this.engine = source["engine"];
	        this.last_analyzed = source["last_analyzed"];
	        this.version = source["version"];
	        this.contexts = this.convertValues(source["contexts"], StoryContext);
	        this.temporal_constraints = this.convertValues(source["temporal_constraints"], TemporalConstraint);
	        this.assertions = this.convertValues(source["assertions"], StoryAssertion);
	        this.narrative_fingerprints = this.convertValues(source["narrative_fingerprints"], NarrativeFingerprint);
	        this.narrative_relations = this.convertValues(source["narrative_relations"], NarrativeFingerprintRelation);
	        this.promotion_stats = this.convertValues(source["promotion_stats"], NarrativePromotionStats);
	        this.events = this.convertValues(source["events"], FingerprintEvent);
	        this.states = this.convertValues(source["states"], StoryStateInterval);
	        this.threads = this.convertValues(source["threads"], StoryThread);
	        this.diagnostics = this.convertValues(source["diagnostics"], FingerprintDiagnostic);
	        this.profiles = this.convertValues(source["profiles"], StoryProfile);
	        this.voices = this.convertValues(source["voices"], CharacterVoiceProfile);
	        this.structure = this.convertValues(source["structure"], StoryStructure);
	        this.author_model = this.convertValues(source["author_model"], StoryAuthorModel);
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
	    fingerprint?: StoryFingerprint;
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
	        this.fingerprint = this.convertValues(source["fingerprint"], StoryFingerprint);
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
	    plot_walker_enabled: boolean;
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
	    read_aloud_cast?: ReadAloudCast;
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
	
	
	
	export class FingerprintQueryAnswer {
	    success: boolean;
	    error?: string;
	    interpretation?: string;
	    answer?: string;
	    events?: FingerprintEvent[];
	    states?: StoryStateInterval[];
	    threads?: StoryThread[];
	    diagnostics?: FingerprintDiagnostic[];
	    voices?: CharacterVoiceProfile[];
	    evidence_ids?: string[];
	    confidence: number;
	
	    static createFrom(source: any = {}) {
	        return new FingerprintQueryAnswer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.error = source["error"];
	        this.interpretation = source["interpretation"];
	        this.answer = source["answer"];
	        this.events = this.convertValues(source["events"], FingerprintEvent);
	        this.states = this.convertValues(source["states"], StoryStateInterval);
	        this.threads = this.convertValues(source["threads"], StoryThread);
	        this.diagnostics = this.convertValues(source["diagnostics"], FingerprintDiagnostic);
	        this.voices = this.convertValues(source["voices"], CharacterVoiceProfile);
	        this.evidence_ids = source["evidence_ids"];
	        this.confidence = source["confidence"];
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
	export class FingerprintQueryRequest {
	    query: string;
	    limit?: number;
	
	    static createFrom(source: any = {}) {
	        return new FingerprintQueryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.limit = source["limit"];
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
	
	
	
	
	
	
	
	
	
	

}

