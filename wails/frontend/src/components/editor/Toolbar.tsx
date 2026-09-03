import type { Editor } from '@tiptap/react'
import { EDITOR_FONTS, EDITOR_FONT_SIZES } from '../../types/draftline'

interface Props {
  editor: Editor | null
}

export default function Toolbar({ editor }: Props) {
  if (!editor) return <div className="editor-toolbar" />

  const ed = editor

  const currentFont = ed.getAttributes('textStyle').fontFamily || 'Merriweather'
  const currentSize = ed.getAttributes('textStyle').fontSize || '14'

  function setFont(font: string) {
    ed.chain().focus().setFontFamily(font).run()
  }

  function setSize(size: string) {
    ed.chain().focus().setFontSize(size).run()
  }

  function setStyle(value: string) {
    if (value === 'blockquote') {
      if (ed.isActive('heading')) ed.chain().focus().setParagraph().run()
      ed.chain().focus().toggleBlockquote().run()
    } else if (value === 'normal') {
      if (ed.isActive('blockquote')) ed.chain().focus().toggleBlockquote().run()
      else ed.chain().focus().setParagraph().run()
    } else {
      if (ed.isActive('blockquote')) ed.chain().focus().toggleBlockquote().run()
      const level = parseInt(value.replace('h', '')) as 1 | 2 | 3
      ed.chain().focus().toggleHeading({ level }).run()
    }
  }

  const styleValue = ed.isActive('heading', { level: 1 }) ? 'h1'
    : ed.isActive('heading', { level: 2 }) ? 'h2'
    : ed.isActive('heading', { level: 3 }) ? 'h3'
    : ed.isActive('blockquote') ? 'blockquote'
    : 'normal'

  return (
    <div className="editor-toolbar">
      {/* Font family */}
      <select className="toolbar-select" style={{ width: 140 }} value={currentFont} onChange={e => setFont(e.target.value)} title="Font family">
        {EDITOR_FONTS.map(f => <option key={f} value={f}>{f}</option>)}
      </select>

      {/* Font size */}
      <select className="toolbar-select" style={{ width: 56 }} value={currentSize} onChange={e => setSize(e.target.value)} title="Font size">
        {EDITOR_FONT_SIZES.map(s => <option key={s} value={s}>{s}</option>)}
      </select>

      <div className="toolbar-sep" />

      {/* Paragraph style */}
      <select className="toolbar-select" style={{ width: 128 }} value={styleValue} onChange={e => setStyle(e.target.value)} title="Paragraph style">
        <option value="normal">Body Text</option>
        <option value="h1">Chapter Title</option>
        <option value="h2">Part Header</option>
        <option value="h3">Scene Heading</option>
        <option value="blockquote">Block Quote</option>
      </select>

      <div className="toolbar-sep" />

      {/* Bold */}
      <button className={`toolbar-btn ${ed.isActive('bold') ? 'active' : ''}`} onClick={() => ed.chain().focus().toggleBold().run()} title="Bold (Ctrl+B)">
        <strong>B</strong>
      </button>

      {/* Italic */}
      <button className={`toolbar-btn ${ed.isActive('italic') ? 'active' : ''}`} onClick={() => ed.chain().focus().toggleItalic().run()} title="Italic (Ctrl+I)" style={{ fontStyle: 'italic' }}>
        I
      </button>

      {/* Underline */}
      <button className={`toolbar-btn ${ed.isActive('underline') ? 'active' : ''}`} onClick={() => ed.chain().focus().toggleUnderline().run()} title="Underline (Ctrl+U)" style={{ textDecoration: 'underline' }}>
        U
      </button>

      {/* Strikethrough */}
      <button className={`toolbar-btn ${ed.isActive('strike') ? 'active' : ''}`} onClick={() => ed.chain().focus().toggleStrike().run()} title="Strikethrough" style={{ textDecoration: 'line-through' }}>
        S
      </button>

      <div className="toolbar-sep" />

      {/* Subscript */}
      <button className={`toolbar-btn ${ed.isActive('subscript') ? 'active' : ''}`} onClick={() => ed.chain().focus().toggleSubscript().run()} title="Subscript">
        x<sub style={{ fontSize: '0.65em' }}>2</sub>
      </button>

      {/* Superscript */}
      <button className={`toolbar-btn ${ed.isActive('superscript') ? 'active' : ''}`} onClick={() => ed.chain().focus().toggleSuperscript().run()} title="Superscript">
        x<sup style={{ fontSize: '0.65em' }}>2</sup>
      </button>

      <div className="toolbar-sep" />

      {/* Align left */}
      <button className={`toolbar-btn ${ed.isActive({ textAlign: 'left' }) ? 'active' : ''}`} onClick={() => ed.chain().focus().setTextAlign('left').run()} title="Align left">
        <svg width="13" height="11" viewBox="0 0 13 11" fill="currentColor"><rect x="0" y="0" width="13" height="1.5" rx="0.75"/><rect x="0" y="3" width="9" height="1.5" rx="0.75"/><rect x="0" y="6" width="13" height="1.5" rx="0.75"/><rect x="0" y="9" width="7" height="1.5" rx="0.75"/></svg>
      </button>

      {/* Align center */}
      <button className={`toolbar-btn ${ed.isActive({ textAlign: 'center' }) ? 'active' : ''}`} onClick={() => ed.chain().focus().setTextAlign('center').run()} title="Align center">
        <svg width="13" height="11" viewBox="0 0 13 11" fill="currentColor"><rect x="0" y="0" width="13" height="1.5" rx="0.75"/><rect x="2" y="3" width="9" height="1.5" rx="0.75"/><rect x="0" y="6" width="13" height="1.5" rx="0.75"/><rect x="3" y="9" width="7" height="1.5" rx="0.75"/></svg>
      </button>

      {/* Align right */}
      <button className={`toolbar-btn ${ed.isActive({ textAlign: 'right' }) ? 'active' : ''}`} onClick={() => ed.chain().focus().setTextAlign('right').run()} title="Align right">
        <svg width="13" height="11" viewBox="0 0 13 11" fill="currentColor"><rect x="0" y="0" width="13" height="1.5" rx="0.75"/><rect x="4" y="3" width="9" height="1.5" rx="0.75"/><rect x="0" y="6" width="13" height="1.5" rx="0.75"/><rect x="6" y="9" width="7" height="1.5" rx="0.75"/></svg>
      </button>

      {/* Justify */}
      <button className={`toolbar-btn ${ed.isActive({ textAlign: 'justify' }) ? 'active' : ''}`} onClick={() => ed.chain().focus().setTextAlign('justify').run()} title="Justify">
        <svg width="13" height="11" viewBox="0 0 13 11" fill="currentColor"><rect x="0" y="0" width="13" height="1.5" rx="0.75"/><rect x="0" y="3" width="13" height="1.5" rx="0.75"/><rect x="0" y="6" width="13" height="1.5" rx="0.75"/><rect x="0" y="9" width="13" height="1.5" rx="0.75"/></svg>
      </button>

      <div className="toolbar-sep" />

      {/* Bullet list */}
      <button className={`toolbar-btn ${ed.isActive('bulletList') ? 'active' : ''}`} onClick={() => ed.chain().focus().toggleBulletList().run()} title="Bullet list">
        <svg width="13" height="11" viewBox="0 0 13 11" fill="currentColor">
          <circle cx="1.2" cy="1.5" r="1.2"/><rect x="4" y="0.75" width="9" height="1.5" rx="0.75"/>
          <circle cx="1.2" cy="5.5" r="1.2"/><rect x="4" y="4.75" width="9" height="1.5" rx="0.75"/>
          <circle cx="1.2" cy="9.5" r="1.2"/><rect x="4" y="8.75" width="7" height="1.5" rx="0.75"/>
        </svg>
      </button>

      {/* Ordered list */}
      <button className={`toolbar-btn ${ed.isActive('orderedList') ? 'active' : ''}`} onClick={() => ed.chain().focus().toggleOrderedList().run()} title="Ordered list">
        <svg width="13" height="11" viewBox="0 0 13 11" fill="currentColor">
          <text x="0" y="3" fontSize="4" fontFamily="monospace">1.</text>
          <rect x="5" y="0.75" width="8" height="1.5" rx="0.75"/>
          <text x="0" y="7" fontSize="4" fontFamily="monospace">2.</text>
          <rect x="5" y="4.75" width="8" height="1.5" rx="0.75"/>
          <text x="0" y="11" fontSize="4" fontFamily="monospace">3.</text>
          <rect x="5" y="8.75" width="6" height="1.5" rx="0.75"/>
        </svg>
      </button>

      {/* Scene break (horizontal rule) */}
      <button
        className="toolbar-btn"
        onClick={() => ed.chain().focus().setHorizontalRule().run()}
        title="Scene break (⁂)"
      >
        <svg width="13" height="5" viewBox="0 0 13 5" fill="currentColor">
          <circle cx="1.5" cy="2.5" r="1.3"/>
          <circle cx="6.5" cy="2.5" r="1.3"/>
          <circle cx="11.5" cy="2.5" r="1.3"/>
        </svg>
      </button>
    </div>
  )
}
