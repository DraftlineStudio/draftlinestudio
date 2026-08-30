import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import TextStyle from '@tiptap/extension-text-style'
import FontFamily from '@tiptap/extension-font-family'
import TextAlign from '@tiptap/extension-text-align'
import { Color } from '@tiptap/extension-color'
import Subscript from '@tiptap/extension-subscript'
import Superscript from '@tiptap/extension-superscript'
import CharacterCount from '@tiptap/extension-character-count'
import { FontSize } from '../../extensions/FontSize'
import { CharacterHighlight } from '../../extensions/CharacterHighlight'
import { SpellCheck } from '../../extensions/SpellCheck'
import { GrammarCheck } from '../../extensions/GrammarCheck'
import { ChapterSearch } from '../../extensions/ChapterSearch'
import { useEffect, useCallback, useState, useRef, useMemo } from 'react'
import Toolbar from './Toolbar'
import ChapterFindReplaceBar from './ChapterFindReplaceBar'
import ContextMenu, { ContextMenuItem } from '../ContextMenu'
import InlinePrompt from './InlinePrompt'
import { checkWord, getDictionaryRoot, getImmediateSuggestions, getSuggestions, isLoaded as isSpellCheckLoaded, normalizeCustomDictionary, setCustomWords, setIgnoredWords, setSpellCheckEnabled } from '../../services/spellCheck'
import { analyzeGrammar, setGrammarCheckEnabled, type GrammarIssue } from '../../services/grammarCheck'
import { useBookStore } from '../../store/bookStore'
import { useEditorStore } from '../../store/editorStore'
import { useAppStore } from '../../store/appStore'
import { isConfirmedCharacter } from '../../utils/characterStatus'

interface ContextMenuState {
  x: number
  y: number
  // Spell check context
  misspelledWord?: string
  wordStart?: number
  wordEnd?: number
  suggestions?: string[]
  suggestionsLoading?: boolean
  grammarIssue?: GrammarIssue & { absoluteFrom: number; absoluteTo: number }
}

interface Props {
  content: string
  onUpdate: (html: string) => void
  chapterLabel?: string
  chapterName?: string
  chapterSubtitle?: string
  onRenameChapter?: (title: string) => void
  onEditSubtitle?: (subtitle: string) => void
}

export default function RichEditor({ content, onUpdate, chapterLabel, chapterName, chapterSubtitle, onRenameChapter, onEditSubtitle }: Props) {
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const [editingTitle, setEditingTitle] = useState(false)
  const [editingSubtitle, setEditingSubtitle] = useState(false)
  const [titleValue, setTitleValue] = useState(chapterName || '')
  const [subtitleValue, setSubtitleValue] = useState(chapterSubtitle || '')
  const [showAiDisabledModal, setShowAiDisabledModal] = useState(false)
  const titleInputRef = useRef<HTMLInputElement>(null)
  const subtitleInputRef = useRef<HTMLInputElement>(null)
  const { settings, openSettings, saveSettings } = useAppStore()

  useEffect(() => {
    const normalizedDictionary = normalizeCustomDictionary(settings.custom_dictionary)
    setCustomWords(normalizedDictionary)
    if (
      normalizedDictionary.length !== settings.custom_dictionary.length
      || normalizedDictionary.some((word, index) => word !== settings.custom_dictionary[index])
    ) {
      void saveSettings({ custom_dictionary: normalizedDictionary })
    }
    setSpellCheckEnabled(settings.spell_check_enabled)
    setGrammarCheckEnabled(settings.grammar_check_enabled)
  }, [settings.custom_dictionary, settings.spell_check_enabled, settings.grammar_check_enabled, saveSettings])

  const handleUpdate = useCallback(
    ({ editor }: { editor: ReturnType<typeof useEditor> & { getHTML: () => string } }) => {
      onUpdate(editor.getHTML())
    },
    [onUpdate],
  )

  // Get character names for highlighting (reads from store on each check)
  const getHighlightedCharacterNames = useBookStore(s => s.getHighlightedCharacterNames)

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ history: { depth: 100 } }),
      Underline,
      TextStyle,
      FontFamily.configure({ types: ['textStyle'] }),
      FontSize,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      Color,
      Subscript,
      Superscript,
      CharacterCount,
      SpellCheck,
      GrammarCheck,
      ChapterSearch,
      CharacterHighlight.configure({
        getNames: () => getHighlightedCharacterNames(),
        highlightClass: 'character-highlight',
      }),
    ],
    content,
    onUpdate: handleUpdate as any,
    editorProps: {
      attributes: {
        // SpellCheck renders persistent TipTap decorations. Native WebView
        // markers disappear whenever our custom context menu changes focus.
        spellcheck: 'false',
      },
    },
  })

  // Expose editor ref to store for selection access
  const setEditorRef = useBookStore(s => s.setEditorRef)
  useEffect(() => {
    if (editor) {
      setEditorRef(editor as any)
    }
    return () => setEditorRef(null)
  }, [editor, setEditorRef])

  // Sync external content changes (chapter switch)
  useEffect(() => {
    if (!editor) return
    const currentHTML = editor.getHTML()
    if (currentHTML !== content) {
      editor.commands.setContent(content || '<p></p>', false)
    }
  }, [content, editor])

  // Force decoration recalculation when highlighted character or its aliases change
  const highlightedCharacterId = useBookStore(s => s.highlightedCharacterId)
  const characters = useBookStore(s => s.book?.story_bible?.characters)

  const confirmedCharacterNames = useMemo(() => (characters ?? [])
    .filter(isConfirmedCharacter)
    .flatMap(character => [character.name, ...(character.aliases ?? [])]), [characters])

  useEffect(() => {
    setIgnoredWords(confirmedCharacterNames)
  }, [confirmedCharacterNames])

  // Compute highlighted names from raw store data (avoids infinite loop from function call in selector)
  const highlightedNames = useMemo(() => {
    if (!highlightedCharacterId || !characters) return []
    const char = characters.find(c => c.id === highlightedCharacterId)
    if (!char) return []
    const names = [char.name]
    if (char.aliases) names.push(...char.aliases)
    return names
  }, [highlightedCharacterId, characters])

  useEffect(() => {
    if (!editor) return
    // Dispatch empty transaction to trigger decoration rebuild
    editor.view.dispatch(editor.state.tr)
  }, [editor, highlightedCharacterId, highlightedNames])

  // Sync title/subtitle values when chapter changes
  useEffect(() => {
    setTitleValue(chapterName || '')
    setSubtitleValue(chapterSubtitle || '')
  }, [chapterName, chapterSubtitle])

  // Focus input when editing starts
  useEffect(() => {
    if (editingTitle && titleInputRef.current) {
      titleInputRef.current.focus()
      titleInputRef.current.select()
    }
  }, [editingTitle])

  useEffect(() => {
    if (editingSubtitle && subtitleInputRef.current) {
      subtitleInputRef.current.focus()
      subtitleInputRef.current.select()
    }
  }, [editingSubtitle])

  // Inline AI prompt state — subscribe to editorStore directly; the bookStore
  // bridge getter does not notify bookStore subscribers when editorStore changes
  const inlinePrompt = useEditorStore(s => s.inlinePrompt)
  const openInlinePrompt = useEditorStore(s => s.openInlinePrompt)

  // Ctrl+L to open inline prompt (if AI is enabled)
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'l') {
        e.preventDefault()
        if (!editor || inlinePrompt) return

        // Check if AI is enabled
        if (!settings.ai_enabled) {
          setShowAiDisabledModal(true)
          return
        }

        const { from } = editor.state.selection
        openInlinePrompt(from)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [editor, inlinePrompt, openInlinePrompt, settings.ai_enabled])

  // Get context around cursor for inline prompt
  const getInlineContext = useCallback(() => {
    if (!editor || !inlinePrompt) return { before: '', after: '' }

    const { doc } = editor.state
    const pos = inlinePrompt.cursorPos
    const docSize = doc.content.size

    // Get ~500 chars before and ~300 chars after
    const beforeStart = Math.max(0, pos - 500)
    const afterEnd = Math.min(docSize, pos + 300)

    const before = doc.textBetween(beforeStart, pos, '\n\n')
    const after = doc.textBetween(pos, afterEnd, '\n\n')

    return { before, after }
  }, [editor, inlinePrompt])

  // Handle inserting generated content
  const handleInlineInsert = useCallback((html: string) => {
    if (!editor || !inlinePrompt) return

    editor
      .chain()
      .focus()
      .setTextSelection(inlinePrompt.cursorPos)
      .insertContent(html)
      .run()

    // Trigger content update
    onUpdate(editor.getHTML())
  }, [editor, inlinePrompt, onUpdate])

  // Handle context menu on editor
  function handleContextMenu(e: React.MouseEvent) {
    e.preventDefault()

    const menuState: ContextMenuState = { x: e.clientX, y: e.clientY }
    let suggestionRequest: { word: string; wordStart: number; wordEnd: number } | null = null

    // Resolve writing diagnostics under the pointer, rather than wherever the
    // keyboard selection happened to be before the context menu opened.
    if (editor) {
      const clickedPosition = editor.view.posAtCoords({ left: e.clientX, top: e.clientY })
      const from = clickedPosition?.pos ?? editor.state.selection.from
      const $pos = editor.state.doc.resolve(from)
      const textNode = $pos.parent

      if (textNode.isTextblock) {
        const text = textNode.textContent
        const nodeStart = $pos.start()
        const offsetInNode = from - nodeStart

        if (settings.grammar_check_enabled) {
          const grammarIssue = analyzeGrammar(text).find(
            issue => offsetInNode >= issue.from && offsetInNode <= issue.to,
          )
          if (grammarIssue) {
            menuState.grammarIssue = {
              ...grammarIssue,
              absoluteFrom: nodeStart + grammarIssue.from,
              absoluteTo: nodeStart + grammarIssue.to,
            }
          }
        }

        if (settings.spell_check_enabled && isSpellCheckLoaded()) {
          let wordStart = offsetInNode
          let wordEnd = offsetInNode

          const isWordCharacter = (character: string) => /[A-Za-z0-9'’‘ʼ＇]/.test(character)
          while (wordStart > 0 && isWordCharacter(text[wordStart - 1])) wordStart--
          while (wordEnd < text.length && isWordCharacter(text[wordEnd])) wordEnd++

          if (wordStart < wordEnd) {
            const word = text.substring(wordStart, wordEnd)

            if (word.length >= 2 && !checkWord(word)) {
              menuState.misspelledWord = word
              menuState.wordStart = nodeStart + wordStart
              menuState.wordEnd = nodeStart + wordEnd
              menuState.suggestions = getImmediateSuggestions(word, 5)
              menuState.suggestionsLoading = true
              suggestionRequest = {
                word,
                wordStart: menuState.wordStart,
                wordEnd: menuState.wordEnd,
              }
            }
          }
        }
      }
    }

    setContextMenu(menuState)

    // Broad typo-js suggestions run in a worker so they cannot block the menu.
    if (suggestionRequest) {
      const request = suggestionRequest
      window.setTimeout(() => {
        void getSuggestions(request.word, 5).then(suggestions => {
          setContextMenu(current => {
            if (
              current?.misspelledWord !== request.word
              || current.wordStart !== request.wordStart
              || current.wordEnd !== request.wordEnd
            ) return current
            return { ...current, suggestions, suggestionsLoading: false }
          })
        })
      }, 0)
    }
  }

  // Context menu actions
  async function handleCut() {
    if (!editor) return
    const { from, to } = editor.state.selection
    const text = editor.state.doc.textBetween(from, to, '\n')
    await navigator.clipboard.writeText(text)
    editor.commands.deleteSelection()
  }

  async function handleCopy() {
    if (!editor) return
    const { from, to } = editor.state.selection
    const text = editor.state.doc.textBetween(from, to, '\n')
    await navigator.clipboard.writeText(text)
  }

  async function handlePaste() {
    if (!editor) return
    const text = await navigator.clipboard.readText()
    editor.commands.insertContent(text)
  }

  // Title editing
  function handleTitleClick() {
    if (onRenameChapter && chapterName) {
      setTitleValue(chapterName)
      setEditingTitle(true)
    }
  }

  function handleTitleSubmit() {
    const trimmed = titleValue.trim()
    if (trimmed && trimmed !== chapterName && onRenameChapter) {
      onRenameChapter(trimmed)
    }
    setEditingTitle(false)
  }

  // Subtitle editing
  function handleSubtitleClick() {
    if (onEditSubtitle) {
      setSubtitleValue(chapterSubtitle || '')
      setEditingSubtitle(true)
    }
  }

  function handleSubtitleSubmit() {
    if (onEditSubtitle) {
      onEditSubtitle(subtitleValue.trim())
    }
    setEditingSubtitle(false)
  }

  const hasSelection = editor && !editor.state.selection.empty

  // Replace misspelled word with suggestion
  function handleSpellSuggestion(suggestion: string) {
    if (!editor || contextMenu?.wordStart == null || contextMenu.wordEnd == null) return
    editor
      .chain()
      .focus()
      .setTextSelection({ from: contextMenu.wordStart, to: contextMenu.wordEnd })
      .insertContent(suggestion)
      .run()
  }

  function handleGrammarSuggestion(
    issue: GrammarIssue & { absoluteFrom: number; absoluteTo: number },
    replacement: string,
  ) {
    if (!editor) return
    let from = issue.absoluteFrom
    if (replacement === '' && from > 1 && /\s/.test(editor.state.doc.textBetween(from - 1, from))) {
      from--
    }
    editor.chain()
      .focus()
      .setTextSelection({ from, to: issue.absoluteTo })
      .insertContent(replacement)
      .run()
  }

  async function handleAddToDictionary(word: string) {
    const normalized = getDictionaryRoot(word.trim())
    if (!normalized) return
    const currentDictionary = normalizeCustomDictionary(settings.custom_dictionary)
    const alreadyAdded = currentDictionary.some(existing =>
      existing.toLocaleLowerCase() === normalized.toLocaleLowerCase(),
    )
    if (alreadyAdded) return

    const customDictionary = normalizeCustomDictionary([...currentDictionary, normalized])
    setCustomWords(customDictionary)
    await saveSettings({ custom_dictionary: customDictionary })
  }

  // Build context menu items
  const contextMenuItems: ContextMenuItem[] = []

  // Add spell suggestions if there's a misspelled word
  if (contextMenu?.misspelledWord) {
    if (contextMenu.suggestions && contextMenu.suggestions.length > 0) {
      contextMenu.suggestions.forEach((suggestion) => {
        contextMenuItems.push({
          label: suggestion,
          icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 013 3L7 19l-4 1 1-4L16.5 3.5z"/></svg>,
          onClick: () => handleSpellSuggestion(suggestion),
        })
      })
    }
    if (contextMenu.suggestionsLoading) {
      contextMenuItems.push({ label: 'Finding more suggestions…', onClick: () => {}, disabled: true })
    } else if (!contextMenu.suggestions?.length) {
      contextMenuItems.push({ label: 'No suggestions', onClick: () => {}, disabled: true })
    }
    contextMenuItems.push({
      label: `Add “${getDictionaryRoot(contextMenu.misspelledWord)}” to dictionary`,
      onClick: () => handleAddToDictionary(contextMenu.misspelledWord!),
    })
    contextMenuItems.push({ label: '', onClick: () => {}, separator: true })
  }

  if (contextMenu?.grammarIssue) {
    const issue = contextMenu.grammarIssue
    contextMenuItems.push({
      label: issue.message,
      onClick: () => {},
      disabled: true,
    })
    issue.replacements.forEach(replacement => {
      contextMenuItems.push({
        label: replacement ? `Replace with “${replacement}”` : 'Remove repeated word',
        icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m5 12 4 4L19 6"/></svg>,
        onClick: () => handleGrammarSuggestion(issue, replacement),
      })
    })
    contextMenuItems.push({ label: '', onClick: () => {}, separator: true })
  }

  // Standard edit actions
  contextMenuItems.push(
    {
      label: 'Cut',
      icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="6" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><line x1="20" y1="4" x2="8.12" y2="15.88"/><line x1="14.47" y1="14.48" x2="20" y2="20"/><line x1="8.12" y1="8.12" x2="12" y2="12"/></svg>,
      onClick: handleCut,
      disabled: !hasSelection,
    },
    {
      label: 'Copy',
      icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>,
      onClick: handleCopy,
      disabled: !hasSelection,
    },
    {
      label: 'Paste',
      icon: <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M16 4h2a2 2 0 012 2v14a2 2 0 01-2 2H6a2 2 0 01-2-2V6a2 2 0 012-2h2"/><rect x="8" y="2" width="8" height="4" rx="1" ry="1"/></svg>,
      onClick: handlePaste,
    },
  )

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Toolbar editor={editor} />
      <ChapterFindReplaceBar editor={editor} />
      <div className="editor-scroll" onContextMenu={handleContextMenu} data-context-menu>
        <div className="editor-content-wrapper">
          {(chapterLabel || chapterName) && (
            <div className="editor-page-header">
              {chapterLabel && <span className="editor-page-chapter-title">{chapterLabel}</span>}
              {chapterName && (
                editingTitle ? (
                  <input
                    ref={titleInputRef}
                    className="editor-page-chapter-name-input"
                    value={titleValue}
                    onChange={e => setTitleValue(e.target.value)}
                    onBlur={handleTitleSubmit}
                    onKeyDown={e => {
                      if (e.key === 'Enter') handleTitleSubmit()
                      if (e.key === 'Escape') setEditingTitle(false)
                    }}
                  />
                ) : (
                  <span
                    className="editor-page-chapter-name editable"
                    onClick={handleTitleClick}
                    title="Click to edit chapter name"
                  >
                    {chapterName}
                  </span>
                )
              )}
              {(chapterSubtitle || onEditSubtitle) && (
                editingSubtitle ? (
                  <input
                    ref={subtitleInputRef}
                    className="editor-page-chapter-subtitle-input"
                    value={subtitleValue}
                    onChange={e => setSubtitleValue(e.target.value)}
                    onBlur={handleSubtitleSubmit}
                    onKeyDown={e => {
                      if (e.key === 'Enter') handleSubtitleSubmit()
                      if (e.key === 'Escape') setEditingSubtitle(false)
                    }}
                    placeholder="Enter subtitle..."
                  />
                ) : (
                  <span
                    className="editor-page-chapter-subtitle editable"
                    onClick={handleSubtitleClick}
                    title={chapterSubtitle ? "Click to edit subtitle" : "Click to add subtitle"}
                  >
                    {chapterSubtitle || '+ Add subtitle'}
                  </span>
                )
              )}
            </div>
          )}
          <div className="editor-page-body">
            <EditorContent editor={editor} />
          </div>
        </div>
        {inlinePrompt && (
          <InlinePrompt
            onInsert={handleInlineInsert}
            onCancel={() => editor?.commands.focus()}
            beforeContext={getInlineContext().before}
            afterContext={getInlineContext().after}
          />
        )}
      </div>
      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={contextMenuItems}
          onClose={() => setContextMenu(null)}
        />
      )}
      {showAiDisabledModal && (
        <div className="dialog-overlay" onClick={() => setShowAiDisabledModal(false)}>
          <div className="dialog ai-disabled-dialog" onClick={e => e.stopPropagation()}>
            <div className="dialog-title">AI Features Disabled</div>
            <p className="ai-disabled-message">
              AI features are currently disabled. Enable them in Settings to use inline AI generation, rewriting, and other AI-powered tools.
            </p>
            <div className="dialog-actions">
              <button className="ai-link-btn" onClick={() => setShowAiDisabledModal(false)}>
                Dismiss
              </button>
              <button
                className="ai-run-btn"
                onClick={() => { setShowAiDisabledModal(false); openSettings() }}
              >
                Open Settings
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
