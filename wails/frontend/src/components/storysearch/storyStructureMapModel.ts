import { types } from '../../../wailsjs/go/models'
import type { EvidenceRecord } from '../../types/draftline'
import type { EraKind, MapCheckpointMark, MapDangle, MapEra, MapLoop, MapNode, MapTick } from './storyMapModel'
import { buildLoop, truncateLabel } from './storyMapModel'

export type StructureProjection = 'story' | 'manuscript'
export type StructureZoom = 'overview' | 'sequence' | 'scene' | 'event'

type Term = { text: string; label: string }
type StoryTime = { context_id?: string; label?: string; day_offset?: number; earliest_day?: number; latest_day?: number; precision?: string; confidence?: number }
type Aggregate = {
  id: string; summary?: string; label?: string; evidence_ids?: string[]; event_ids?: string[]
  fingerprint_event_ids?: string[]; sequence_ids?: string[]; scene_ids?: string[]; thread_ids?: string[]
  character_ids?: string[]; character_names?: string[]; locations?: Term[]; objects?: Term[]
  context_id?: string; context_ids?: string[]; story_time?: StoryTime; narrative_order?: number
  narrative_start?: number; narrative_end?: number; salience?: number; confidence?: number
  chapter_title?: string; section?: string; section_index?: number; kinds?: string[]
  state_ids?: string[]; obligation_thread_ids?: string[]; convergence_event_ids?: string[]
}
type Structure = {
  significant_events?: Aggregate[]; scenes?: Aggregate[]; sequences?: Aggregate[]
  narrative_threads?: Aggregate[]; arcs?: Aggregate[]
}

export interface StructureMapLayout {
  eras: MapEra[]; ticks: MapTick[]; nodes: MapNode[]; pathD: string; threadPaths: { id: string; d: string }[]
  checkpoints: MapCheckpointMark[]; dangles: MapDangle[]; loop: MapLoop | null
  width: number; height: number; laneCount: number; hiddenCount: number
  aggregateById: Map<string, Aggregate>; levelLabel: string
}

const MAX_BY_ZOOM: Record<StructureZoom, number> = { overview: 36, sequence: 64, scene: 96, event: 160 }
const WIDTH_PER_NODE: Record<StructureZoom, number> = { overview: 74, sequence: 58, scene: 44, event: 34 }

export function structureOf(fingerprint: types.StoryFingerprint): Structure | null {
  return ((fingerprint as unknown as { structure?: Structure }).structure ?? null)
}

function aggregatePool(structure: Structure, zoom: StructureZoom): Aggregate[] {
  if (zoom === 'overview') return overviewCoverage(structure)
  if (zoom === 'sequence') return (structure.sequences ?? []).map(item => hydrate(item, structure))
  if (zoom === 'scene') return (structure.scenes ?? []).map(item => hydrate(item, structure))
  return structure.significant_events ?? []
}

function hydrate(item: Aggregate, structure: Structure): Aggregate {
  const children = (item.event_ids ?? []).map(id => (structure.significant_events ?? []).find(event => event.id === id)).filter((event): event is Aggregate => !!event)
  const firstChild = children[0]
  return {
    ...item,
    context_id: item.context_id ?? firstChild?.context_id,
    story_time: item.story_time ?? firstChild?.story_time,
    chapter_title: item.chapter_title ?? firstChild?.chapter_title,
    section: item.section ?? firstChild?.section,
    section_index: item.section_index ?? firstChild?.section_index,
    fingerprint_event_ids: item.fingerprint_event_ids ?? children.flatMap(child => child.fingerprint_event_ids ?? []),
  }
}

// Coverage precedes salience: every arc, narrative thread, temporal context,
// convergence and plot obligation gets a representative before spare slots
// are filled by independently high-scoring beats.
export function overviewCoverage(structure: Structure): Aggregate[] {
  const events = structure.significant_events ?? []
  const byId = new Map(events.map(event => [event.id, event]))
  const selected = new Map<string, Aggregate>()
  const addEvent = (id: string | undefined) => { const event = id ? byId.get(id) : undefined; if (event) selected.set(event.id, event) }
  const representative = (ids: string[] | undefined) => [...(ids ?? [])].map(id => byId.get(id)).filter((v): v is Aggregate => !!v).sort((a, b) => (b.salience ?? 0) - (a.salience ?? 0))[0]

  for (const arc of structure.arcs ?? []) addEvent(representative(arc.event_ids)?.id)
  for (const thread of structure.narrative_threads ?? []) {
    addEvent(thread.event_ids?.[0])
    addEvent(representative(thread.event_ids)?.id)
    addEvent(thread.event_ids?.[(thread.event_ids?.length ?? 1) - 1])
    for (const id of thread.convergence_event_ids ?? []) addEvent(id)
  }
  const seenContexts = new Set<string>()
  for (const event of events) {
    const context = event.context_id ?? ''
    if (!seenContexts.has(context)) { seenContexts.add(context); addEvent(event.id) }
    if ((event.obligation_thread_ids?.length ?? 0) > 0) addEvent(event.id)
  }
  for (const event of [...events].sort((a, b) => (b.salience ?? 0) - (a.salience ?? 0))) {
    if (selected.size >= MAX_BY_ZOOM.overview) break
    selected.set(event.id, event)
  }
  return [...selected.values()].sort((a, b) => orderOf(a) - orderOf(b))
}

function orderOf(value: Aggregate): number { return value.narrative_order ?? value.narrative_start ?? 0 }
function dayOf(value: Aggregate): number | null {
  if (typeof value.story_time?.day_offset === 'number') return value.story_time.day_offset
  if (typeof value.story_time?.earliest_day === 'number') return value.story_time.earliest_day
  return null
}

function contextKind(fp: types.StoryFingerprint, aggregate: Aggregate): EraKind {
  const id = aggregate.context_id ?? aggregate.context_ids?.[0]
  const kind = (fp.contexts ?? []).find(context => context.id === id)?.kind
  return ['primary', 'past', 'memory', 'dream', 'simulation'].includes(kind ?? '') ? kind as EraKind : 'unknown'
}

function requiredEvents(structure: Structure): Set<string> {
  const result = new Set<string>()
  for (const thread of structure.narrative_threads ?? []) for (const id of thread.convergence_event_ids ?? []) result.add(id)
  for (const event of structure.significant_events ?? []) if ((event.obligation_thread_ids?.length ?? 0) > 0) result.add(event.id)
  return result
}

export function buildStructureMapLayout(fp: types.StoryFingerprint, projection: StructureProjection, zoom: StructureZoom): StructureMapLayout | null {
  const structure = structureOf(fp)
  if (!structure) return null
  const all = aggregatePool(structure, zoom)
  const max = MAX_BY_ZOOM[zoom]
  const mandatory = requiredEvents(structure)
  let visible = all
  if (all.length > max) {
    const keep = new Map<string, Aggregate>()
    for (const item of all) if (mandatory.has(item.id)) keep.set(item.id, item)
    for (const item of [...all].sort((a, b) => (b.salience ?? 0) - (a.salience ?? 0))) { if (keep.size >= max) break; keep.set(item.id, item) }
    visible = [...keep.values()]
  }
  visible.sort((a, b) => projection === 'story'
    ? ((dayOf(a) ?? Number.MAX_SAFE_INTEGER) - (dayOf(b) ?? Number.MAX_SAFE_INTEGER) || orderOf(a) - orderOf(b))
    : orderOf(a) - orderOf(b))

  const threads = structure.narrative_threads ?? []
  const laneByEvent = new Map<string, number>()
  threads.slice(0, 7).forEach((thread, lane) => { for (const id of thread.event_ids ?? []) if (!laneByEvent.has(id)) laneByEvent.set(id, lane) })
  const laneCount = Math.max(1, Math.min(8, threads.length || 1))
  const height = 104 + laneCount * 54
  const width = Math.max(980, 170 + visible.length * WIDTH_PER_NODE[zoom])
  const aggregateById = new Map(visible.map(item => [item.id, item]))
  const nodes: MapNode[] = visible.map((item, index) => {
    const lane = zoom === 'overview' ? (laneByEvent.get(item.id) ?? 0) : 0
    const kind = contextKind(fp, item)
    const day = dayOf(item)
    const uncertain = projection === 'story' && (day === null || (item.story_time?.confidence ?? item.confidence ?? 0) < .7)
    return { id: item.id, x: 54 + index * ((width - 108) / Math.max(1, visible.length - 1)), y: 62 + lane * 54, r: (item.salience ?? 0) >= .72 ? 8 : 6.5, lane, ord: index + 1, label: truncateLabel(item.summary || item.label || 'Story beat'), kind, eraId: item.context_id ?? kind, dashed: uncertain, major: (item.salience ?? 0) >= .72, narrativeOrder: orderOf(item) }
  })
  let pathD = ''
  if (nodes.length) { pathD = `M ${nodes[0].x} ${nodes[0].y}`; for (let i = 1; i < nodes.length; i++) { const a = nodes[i - 1], b = nodes[i], mid = (a.x + b.x) / 2; pathD += ` C ${mid} ${a.y}, ${mid} ${b.y}, ${b.x} ${b.y}` } }
  const threadPaths = zoom === 'overview' ? threads.map(thread => {
    const members = (thread.event_ids ?? []).map(id => nodes.find(node => node.id === id)).filter((node): node is MapNode => !!node)
    if (members.length < 2) return { id: thread.id, d: '' }
    let d = `M ${members[0].x} ${members[0].y}`
    for (let i = 1; i < members.length; i++) { const a = members[i - 1], b = members[i], mid = (a.x + b.x) / 2; d += ` C ${mid} ${a.y}, ${mid} ${b.y}, ${b.x} ${b.y}` }
    return { id: thread.id, d }
  }).filter(path => path.d) : []

  const grouped = new Map<string, Aggregate[]>()
  for (const item of visible) { const key = projection === 'manuscript' ? (item.chapter_title || 'Manuscript') : dayOf(item) === null ? 'Floating / relative' : `Story day ${dayOf(item)}`; grouped.set(key, [...(grouped.get(key) ?? []), item]) }
  const eras: MapEra[] = []
  for (const [label, items] of grouped) { const itemNodes = items.map(item => nodes.find(n => n.id === item.id)).filter((n): n is MapNode => !!n); if (!itemNodes.length) continue; const x = Math.min(...itemNodes.map(n => n.x)) - 22; const end = Math.max(...itemNodes.map(n => n.x)) + 22; eras.push({ id: label, contextId: items[0].context_id ?? '', label: label.toUpperCase(), sub: `${items.length} ${zoom}${items.length === 1 ? '' : 's'}`, kind: contextKind(fp, items[0]), anomaly: false, x, width: Math.max(64, end - x), eventIds: items.map(item => item.id) }) }
  const ticks = eras.map(era => ({ x: era.x, label: era.label }))
  const fpNodes = new Map<string, MapNode>()
  for (const node of nodes) { const aggregate = aggregateById.get(node.id); for (const id of aggregate?.fingerprint_event_ids ?? aggregate?.event_ids ?? []) fpNodes.set(id, node) }
  return { eras, ticks, nodes, pathD, threadPaths, checkpoints: [], dangles: [], loop: buildLoop(fp, fpNodes, height), width, height, laneCount, hiddenCount: all.length - visible.length, aggregateById, levelLabel: zoom }
}

export interface StructureDetail {
  id: string; fingerprintEventId: string; fingerprintEventIds: string[]; kindChip: string; kind: EraKind; when: string; title: string
  location: string; quotes: EvidenceRecord[]; stateChanges: string[]; obligations: string[]; chips: string[]
  confidence: number; salience: number; navigation: { section: string; sectionIndex: number; query: string } | null
}

export function describeAggregate(fp: types.StoryFingerprint, evidence: Map<string, EvidenceRecord>, layout: StructureMapLayout, id: string): StructureDetail | null {
  const aggregate = layout.aggregateById.get(id)
  if (!aggregate) return null
  const evidenceIds = aggregate.evidence_ids ?? []
  const quotes = evidenceIds.map(evidenceId => evidence.get(evidenceId)).filter((item): item is EvidenceRecord => !!item)
  const firstRecord = quotes[0]
  const structure = structureOf(fp)
  const significantById = new Map((structure?.significant_events ?? []).map(event => [event.id, event]))
  const fpEventIds = aggregate.fingerprint_event_ids ?? (aggregate.event_ids ?? []).flatMap(eventId => significantById.get(eventId)?.fingerprint_event_ids ?? [])
  const firstFp = (fp.events ?? []).find(event => fpEventIds.includes(event.id))
  const kind = contextKind(fp, aggregate)
  const states = (fp.states ?? []).filter(state => fpEventIds.includes(state.start_event_id)).map(state => `${state.kind}: ${state.entity_name || state.entity_id} — ${state.value}`)
  const obligations = (fp.threads ?? []).filter(thread => (thread.event_ids ?? []).some(eventId => fpEventIds.includes(eventId))).map(thread => `${thread.label} — ${thread.state}`)
  const day = dayOf(aggregate)
  return { id, fingerprintEventId: firstFp?.id ?? '', fingerprintEventIds: fpEventIds, kindChip: `${layout.levelLabel} · ${kind}`, kind, when: aggregate.story_time?.label || (day === null ? 'Floating / relative placement' : `Story day ${day}`), title: aggregate.summary || aggregate.label || 'Story structure', location: [firstFp?.chapter_title, ...(aggregate.locations ?? []).map(term => term.text)].filter(Boolean).join(' · '), quotes, stateChanges: states, obligations, chips: aggregate.character_names ?? [], confidence: aggregate.confidence ?? 0, salience: aggregate.salience ?? 0, navigation: firstRecord ? { section: firstRecord.section, sectionIndex: firstRecord.section_index, query: firstRecord.text.slice(0, 80) } : null }
}
