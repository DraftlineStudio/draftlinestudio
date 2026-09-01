import { types } from '../../../wailsjs/go/models'

interface Props {
  insight: types.StorySearchInsight
  total: number
  truncated: boolean
  shown: number
  onFirst: () => void
  onLast: () => void
  onRelated: (term: string) => void
}

export default function DetailInsightPanel({ insight, total, truncated, shown, onFirst, onLast, onRelated }: Props) {
  return (
    <section className="detail-insight" aria-label="Story detail trail">
      <div className="detail-insight-lede">
        <div>
          <span className="detail-insight-eyebrow">Story trail</span>
          <strong>{trailHeadline(total, insight.chapter_count)}</strong>
          {insight.interpreted_query && <small>Interpreted as: {insight.interpreted_query}</small>}
        </div>
        <div className="detail-insight-jump">
          <button type="button" onClick={onFirst}>Earliest</button>
          <button type="button" onClick={onLast}>Latest shown</button>
        </div>
      </div>

      <div className="detail-insight-stats">
        <InsightStat value={total} label={total === 1 ? 'source scene' : 'source scenes'} />
        <InsightStat value={insight.chapter_count} label={insight.chapter_count === 1 ? 'chapter' : 'chapters'} />
        <InsightStat value={insight.event_count} label={insight.event_count === 1 ? 'indexed event' : 'indexed events'} />
        <InsightStat value={insight.fact_count} label={insight.fact_count === 1 ? 'indexed fact' : 'indexed facts'} />
      </div>

      {!!insight.signals?.length && (
        <div className="detail-insight-signals">
          {insight.signals.map((signal, index) => (
            <div className={`detail-insight-signal ${signal.kind}`} key={`${signal.title}-${index}`}>
              <i>{signal.kind === 'answer' ? '✓' : signal.kind === 'attention' ? '!' : 'i'}</i>
              <span><strong>{signal.title}</strong><small>{signal.detail}</small></span>
            </div>
          ))}
        </div>
      )}

      <div className="detail-insight-chapters">
        <span>Chapter trail</span>
        <div>
          {insight.chapters.map(chapter => (
            <span className="detail-insight-chapter" key={chapter.chapter_index} title={`${chapter.evidence_count} indexed evidence records`}>
              <strong>{chapter.chapter_title}</strong>
              <small>{chapter.occurrences} {chapter.occurrences === 1 ? 'scene' : 'scenes'}</small>
            </span>
          ))}
        </div>
      </div>

      {!!insight.related_terms?.length && (
        <div className="detail-insight-related">
          <span>Details traveling with it</span>
          <div>
            {insight.related_terms.map(term => (
              <button type="button" key={`${term.label}-${term.text}`} title={`Trace ${term.text} · ${term.label}`} onClick={() => onRelated(term.text)}>
                {term.text}<small>{term.count}</small>
              </button>
            ))}
          </div>
        </div>
      )}

      {truncated && <div className="detail-insight-limit">Showing the first {shown} source scenes; fingerprint counts cover all {total}.</div>}
    </section>
  )
}

function InsightStat({ value, label }: { value: number; label: string }) {
  return <span><strong>{value}</strong><small>{label}</small></span>
}

function trailHeadline(total: number, chapters: number): string {
  if (total === 1) return 'One precise occurrence in the manuscript'
  if (chapters === 1) return `${total} connected occurrences in one chapter`
  return `${total} connected occurrences across ${chapters} chapters`
}
