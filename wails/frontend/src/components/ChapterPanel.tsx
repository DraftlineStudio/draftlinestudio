import { useState, useRef, useEffect, useMemo } from 'react'
import { useShallow } from 'zustand/react/shallow'
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
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useBookStore } from '../store/bookStore'
import type { Section } from '../types/draftline'
import { FRONT_MATTER_TYPES, BODY_TYPES, BACK_MATTER_TYPES } from '../types/draftline'
import ContextMenu, { ContextMenuItem } from './ContextMenu'
import { countBookWords } from '../utils/textUtils'

interface SortableItemProps {
  id: string
  section: Section
  index: number
  title: string
  subtitle?: string
  isSelected: boolean
  onClick: () => void
  onDelete: () => void
  onRename: (title: string) => void
  onEditSubtitle: (subtitle: string) => void
  canDelete: boolean
}

function SortableChapterItem({ id, title, subtitle, isSelected, onClick, onDelete, onRename, onEditSubtitle, canDelete }: SortableItemProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id })
  const [isEditing, setIsEditing] = useState(false)
  const [editValue, setEditValue] = useState(title)
  const [editingSubtitle, setEditingSubtitle] = useState(false)
  const [subtitleValue, setSubtitleValue] = useState(subtitle || '')
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number } | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [isEditing])

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  function handleDoubleClick(e: React.MouseEvent) {
    e.stopPropagation()
    setEditValue(title)
    setIsEditing(true)
  }

  function handleRenameSubmit() {
    const trimmed = editValue.trim()
    if (trimmed && trimmed !== title) {
      onRename(trimmed)
    }
    setIsEditing(false)
  }

  function handleSubtitleSubmit() {
    onEditSubtitle(subtitleValue.trim())
    setEditingSubtitle(false)
  }

  function handleContextMenu(e: React.MouseEvent) {
    e.preventDefault()
    e.stopPropagation()
    setContextMenu({ x: e.clientX, y: e.clientY })
  }

  function handleDeleteClick(e?: React.MouseEvent) {
    e?.stopPropagation()
    setShowDeleteConfirm(true)
  }

  function confirmDelete() {
    onDelete()
    setShowDeleteConfirm(false)
  }

  const contextMenuItems: ContextMenuItem[] = [
    {
      label: 'Rename',
      icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7" /><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z" /></svg>,
      onClick: () => { setEditValue(title); setIsEditing(true) },
    },
    {
      label: subtitle ? 'Edit Subtitle' : 'Add Subtitle',
      icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="4" y1="9" x2="20" y2="9" /><line x1="4" y1="15" x2="14" y2="15" /></svg>,
      onClick: () => { setSubtitleValue(subtitle || ''); setEditingSubtitle(true) },
    },
    {
      label: 'Delete',
      icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="3 6 5 6 21 6" /><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2" /></svg>,
      onClick: handleDeleteClick,
      danger: true,
      disabled: !canDelete,
    },
  ]

  if (isEditing) {
    return (
      <div ref={setNodeRef} style={style} className={`chapter-item ${isSelected ? 'selected' : ''}`}>
        <input
          ref={inputRef}
          className="chapter-rename-input"
          value={editValue}
          onChange={e => setEditValue(e.target.value)}
          onBlur={handleRenameSubmit}
          onKeyDown={e => {
            if (e.key === 'Enter') handleRenameSubmit()
            if (e.key === 'Escape') setIsEditing(false)
          }}
        />
      </div>
    )
  }

  if (editingSubtitle) {
    return (
      <div ref={setNodeRef} style={style} className={`chapter-item ${isSelected ? 'selected' : ''}`}>
        <input
          className="chapter-rename-input"
          value={subtitleValue}
          onChange={e => setSubtitleValue(e.target.value)}
          onBlur={handleSubtitleSubmit}
          onKeyDown={e => {
            if (e.key === 'Enter') handleSubtitleSubmit()
            if (e.key === 'Escape') setEditingSubtitle(false)
          }}
          placeholder="Enter subtitle..."
          autoFocus
        />
      </div>
    )
  }

  return (
    <>
      <div
        ref={setNodeRef}
        style={style}
        className={`chapter-item ${isSelected ? 'selected' : ''} ${isDragging ? 'dragging' : ''} ${subtitle ? 'has-subtitle' : ''}`}
        onClick={onClick}
        onDoubleClick={handleDoubleClick}
        onContextMenu={handleContextMenu}
      >
        <span className="chapter-item-icon">
          <svg width="11" height="13" viewBox="0 0 11 13" fill="none" stroke="currentColor" strokeWidth="1.2">
            <rect x="0.6" y="0.6" width="9.8" height="11.8" rx="1" />
            <line x1="2.5" y1="4" x2="8.5" y2="4" />
            <line x1="2.5" y1="6.5" x2="8.5" y2="6.5" />
            <line x1="2.5" y1="9" x2="6" y2="9" />
          </svg>
        </span>
        <div className="chapter-item-text">
          <span className="chapter-item-title">{title}</span>
          {subtitle && <span className="chapter-item-subtitle">{subtitle}</span>}
        </div>
        {canDelete && (
          <button
            className="chapter-item-delete"
            onClick={handleDeleteClick}
            title="Delete"
          >
            ×
          </button>
        )}
        <span className="chapter-item-drag" {...attributes} {...listeners} title="Drag to reorder">
          ⠿
        </span>
      </div>
      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={contextMenuItems}
          onClose={() => setContextMenu(null)}
        />
      )}
      {showDeleteConfirm && (
        <div className="dialog-overlay" onClick={() => setShowDeleteConfirm(false)}>
          <div className="dialog" onClick={e => e.stopPropagation()} style={{ maxWidth: 360 }}>
            <div className="dialog-title">Delete Chapter?</div>
            <p style={{ margin: '8px 0 16px', fontSize: 12, color: 'var(--text-secondary)' }}>
              Are you sure you want to delete "{title}"? This action cannot be undone.
            </p>
            <div className="dialog-actions">
              <button className="dialog-btn" onClick={() => setShowDeleteConfirm(false)}>Cancel</button>
              <button className="dialog-btn" style={{ background: '#e55', borderColor: '#e55', color: '#fff' }} onClick={confirmDelete}>Delete</button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

interface SectionListProps {
  section: Section
  label: string
  types: string[]
}

function SectionList({ section, label, types: _types }: SectionListProps) {
  const { book, currentSection, currentIndex, setCurrentChapter, deleteChapter, moveChapter, openNewChapterDialog, updateChapterTitle, updateChapterSubtitle } = useBookStore(useShallow(s => ({
    book: s.book,
    currentSection: s.currentSection,
    currentIndex: s.currentIndex,
    setCurrentChapter: s.setCurrentChapter,
    deleteChapter: s.deleteChapter,
    moveChapter: s.moveChapter,
    openNewChapterDialog: s.openNewChapterDialog,
    updateChapterTitle: s.updateChapterTitle,
    updateChapterSubtitle: s.updateChapterSubtitle,
  })))
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
              subtitle={item.subtitle}
              isSelected={currentSection === section && currentIndex === i}
              onClick={() => setCurrentChapter(section, i)}
              onDelete={() => deleteChapter(section, i)}
              onRename={(title) => updateChapterTitle(section, i, title)}
              onEditSubtitle={(subtitle) => updateChapterSubtitle(section, i, subtitle)}
              canDelete={!(section === 'body' && items.length <= 1)}
            />
          ))}
        </SortableContext>
      </DndContext>
    </div>
  )
}

export default function ChapterPanel() {
  const { book, currentSection, setCurrentChapter, leftPanelOpen, toggleLeftPanel } = useBookStore(useShallow(s => ({
    book: s.book,
    currentSection: s.currentSection,
    setCurrentChapter: s.setCurrentChapter,
    leftPanelOpen: s.leftPanelOpen,
    toggleLeftPanel: s.toggleLeftPanel,
  })))

  // Whole-book recount re-parses every chapter's HTML; memoize on the book
  // reference so it only runs when content changes, not on unrelated store
  // updates (isDirty / statusMessage / isAutoSaving) that used to re-render this
  // panel via a bare store subscription.
  const totalWords = useMemo(() => (book ? countBookWords(book) : 0), [book])

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
