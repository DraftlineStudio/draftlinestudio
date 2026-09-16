// The synopsis a chapter's own Planner cards make. Every sentence is a card's
// synopsis text, written by the author; nothing here is generated prose and
// nothing is read out of the manuscript. A chapter the author has edited by
// hand keeps their text instead.

import type { PlannerCard, PlannerData } from '../../types/planner'
import type { PlannerChapter } from './plannerModel'

// One sentence of a chapter's synopsis and the card that wrote it.
export interface SynopsisEntry {
  text: string
  cardId: string
  chapterId: string
  scene?: number
}

// A chapter's synopsis entries, in timeline order: story-line order first,
// then card order within a line.
export function synopsisEntries(planner: PlannerData, chapterId: string): SynopsisEntry[] {
  const laneOrder = new Map(planner.lanes.map((l, i) => [l.id, i]))
  const rank = (c: PlannerCard) => laneOrder.get(c.lines[0]) ?? 99
  return planner.cards
    .filter(c => c.chapter_id === chapterId && c.synopsis.trim())
    .sort((a, b) => rank(a) - rank(b))
    .map(card => ({
      text: card.synopsis.trim(),
      cardId: card.id,
      chapterId: card.link?.chapter_id || chapterId,
      scene: card.link?.scene,
    }))
}

// Each chapter's paragraph is its entries joined in order. Chapters without
// entries stay blank, never invented.
export function generatedSynopsis(planner: PlannerData, chapterId: string): string {
  return synopsisEntries(planner, chapterId).map(e => e.text).join(' ')
}

export function synopsisText(planner: PlannerData, chapterId: string): string {
  const edited = planner.synopsis?.[chapterId]
  return edited !== undefined ? edited : generatedSynopsis(planner, chapterId)
}

// Where one entry's card sits, for the export trace and the tooltip.
export function synopsisTrace(entry: SynopsisEntry, chapters: PlannerChapter[]): string {
  const chapter = chapters.find(c => c.id === entry.chapterId)
  const where = chapter ? `Chapter ${chapter.num}` : 'Later'
  return entry.scene ? `[${where} · Scene ${entry.scene}]` : `[${where}]`
}

// The synopsis as Markdown: one paragraph per chapter, then a trace line per
// entry naming the card behind it. A chapter the author has edited by hand is
// their own text and carries no traces, because the sentences are no longer
// attributable one by one.
export function synopsisMarkdown(planner: PlannerData, chapters: PlannerChapter[]): string {
  return chapters
    .map(ch => ({ ch, text: synopsisText(planner, ch.id), edited: planner.synopsis?.[ch.id] !== undefined }))
    .filter(r => r.text.trim())
    .map(r => {
      const head = `## Chapter ${r.ch.num}${r.ch.title ? ` — ${r.ch.title}` : ''}\n\n${r.text}`
      if (r.edited) return head
      const traces = synopsisEntries(planner, r.ch.id).map(e => `- ${e.text} ${synopsisTrace(e, chapters)}`)
      return traces.length ? `${head}\n\n${traces.join('\n')}` : head
    })
    .join('\n\n')
}
