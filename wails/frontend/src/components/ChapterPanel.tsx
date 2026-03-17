import {
  DndContext,
  closestCenter,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
  arrayMove as _arrayMove,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useBookStore } from '../store/bookStore'
import type { Section } from '../types/draftline'
import { FRONT_MATTER_TYPES, BODY_TYPES, BACK_MATTER_TYPES } from '../types/draftline'

function countWords(html: string): number {
  const div = document.createElement('div')
  div.innerHTML = html
  const text = div.textContent || div.innerText || ''
  return text.trim().split(/\s+/).filter((w) => w.length > 0).length
}

interface SortableItemProps {
  id: string
  section: Section
  index: number
  title: string
  isSelected: boolean
  onClick: () => void
  onDelete: () => void
  canDelete: boolean
}

function SortableChapterItem({ id, title, isSelected, onClick, onDelete, canDelete }: SortableItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id })

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`chapter-item ${isSelected ? 'selected' : ''} ${isDragging ? 'dragging' : ''}`}
      onClick={onClick}
    >
      <span className="chapter-item-icon">
        <svg width="11" height="13" viewBox="0 0 11 13" fill="none" stroke="currentColor" strokeWidth="1.2">
          <rect x="0.6" y="0.6" width="9.8" height="11.8" rx="1" />
          <line x1="2.5" y1="4" x2="8.5" y2="4" />
          <line x1="2.5" y1="6.5" x2="8.5" y2="6.5" />
          <line x1="2.5" y1="9" x2="6" y2="9" />
        </svg>
      </span>
      <span className="chapter-item-title">{title}</span>
      {canDelete && (
        <button
          className="chapter-item-delete"
          onClick={(e) => { e.stopPropagation(); onDelete() }}
          title="Delete"
        >
          ×
        </button>
      )}
      <span className="chapter-item-drag" {...attributes} {...listeners} title="Drag to reorder">
        ⠿
      </span>
    </div>
  )
}

interface SectionListProps {
  section: Section
  label: string
  types: string[]
}

function SectionList({ section, label, types: _types }: SectionListProps) {
  const { book, currentSection, currentIndex, setCurrentChapter, deleteChapter, moveChapter, openNewChapterDialog } = useBookStore()
  if (!book) return null

  const items = section === 'front_matter' ? book.front_matter
    : section === 'body' ? book.body
    : book.back_matter

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const ids = items.map((_, i) => `${section}-${i}`)

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (!over || active.id === over.id) return
    const fromIdx = ids.indexOf(active.id as string)
    const toIdx = ids.indexOf(over.id as string)
    if (fromIdx === -1 || toIdx === -1) return
    moveChapter(section, fromIdx, toIdx)
  }

  return (
    <div className="chapter-section">
      <div className="chapter-section-header">
        <div className="chapter-section-header-left">
          <span className={`chapter-section-dot ${section === 'front_matter' ? 'front' : section === 'body' ? 'body' : 'back'}`} />
          <span className="chapter-section-label">{label}</span>
        </div>
        <button className="chapter-section-add" onClick={() => openNewChapterDialog(section)} title={`Add to ${label}`}>+</button>
      </div>
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={ids} strategy={verticalListSortingStrategy}>
          {items.map((item, i) => (
            <SortableChapterItem
              key={ids[i]}
              id={ids[i]}
              section={section}
              index={i}
              title={item.title}
              isSelected={currentSection === section && currentIndex === i}
              onClick={() => setCurrentChapter(section, i)}
              onDelete={() => deleteChapter(section, i)}
              canDelete={!(section === 'body' && items.length <= 1)}
            />
          ))}
        </SortableContext>
      </DndContext>
    </div>
  )
}

export default function ChapterPanel() {
  const { book, currentSection, currentIndex, setCurrentChapter, leftPanelOpen, toggleLeftPanel } = useBookStore()

  const allContent = book ? [
    book.copyright,
    ...book.front_matter.map((c) => c.content),
    ...book.body.map((c) => c.content),
    ...book.back_matter.map((c) => c.content),
  ] : []
  const totalWords = allContent.reduce((sum, html) => sum + countWords(html || ''), 0)

  return (
    <div className={`chapter-panel${leftPanelOpen ? '' : ' collapsed'}`}>
      {/* Full-height strip shown when panel is collapsed */}
      <button className="panel-collapsed-strip" onClick={toggleLeftPanel} title="Expand Manuscript panel (Ctrl+[)">
        <svg width="6" height="10" viewBox="0 0 6 10" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
          <path d="M1 1l4 4-4 4" />
        </svg>
        <span className="panel-collapsed-label">Manuscript</span>
      </button>

      <div className="chapter-panel-inner">
        <div className="chapter-panel-header">
          <span className="chapter-panel-header-title">Manuscript</span>
          <button className="panel-collapse-btn" onClick={toggleLeftPanel} title="Collapse panel (Ctrl+[)">
            <svg width="6" height="10" viewBox="0 0 6 10" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 1L1 5l4 4" />
            </svg>
          </button>
        </div>

        <div className="chapter-list-scroll">
          {book && <>
            {/* Copyright — single, non-sortable */}
            <div className="chapter-section">
              <div className="chapter-section-header">
                <div className="chapter-section-header-left">
                  <span className="chapter-section-dot front" />
                  <span className="chapter-section-label">Front Pages</span>
                </div>
              </div>
              <div
                className={`chapter-item ${currentSection === 'copyright' ? 'selected' : ''}`}
                onClick={() => setCurrentChapter('copyright', 0)}
              >
                <span className="chapter-item-icon">
                  <svg width="11" height="13" viewBox="0 0 11 13" fill="none" stroke="currentColor" strokeWidth="1.2">
                    <rect x="0.6" y="0.6" width="9.8" height="11.8" rx="1" />
                    <line x1="2.5" y1="4" x2="8.5" y2="4" />
                    <line x1="2.5" y1="6.5" x2="8.5" y2="6.5" />
                    <line x1="2.5" y1="9" x2="6" y2="9" />
                  </svg>
                </span>
                <span className="chapter-item-title">Copyright Page</span>
              </div>
            </div>

            <SectionList section="front_matter" label="Front Matter" types={FRONT_MATTER_TYPES} />
            <SectionList section="body" label="Body" types={BODY_TYPES} />
            <SectionList section="back_matter" label="Back Matter" types={BACK_MATTER_TYPES} />
          </>}
        </div>

        <div className="chapter-panel-footer">
          <div className="chapter-panel-wordcount">{totalWords.toLocaleString()} words total</div>
        </div>
      </div>
    </div>
  )
}
