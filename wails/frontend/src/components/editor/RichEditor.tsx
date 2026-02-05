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
import { useEffect, useCallback } from 'react'
import Toolbar from './Toolbar'

interface Props {
  content: string
  onUpdate: (html: string) => void
  chapterLabel?: string
  chapterName?: string
}

export default function RichEditor({ content, onUpdate, chapterLabel, chapterName }: Props) {
  const handleUpdate = useCallback(
    ({ editor }: { editor: ReturnType<typeof useEditor> & { getHTML: () => string } }) => {
      onUpdate(editor.getHTML())
    },
    [onUpdate],
  )

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
    ],
    content,
    onUpdate: handleUpdate as any,
    editorProps: {
      attributes: {
        spellcheck: 'true',
      },
    },
  })

  // Sync external content changes (chapter switch)
  useEffect(() => {
    if (!editor) return
    const currentHTML = editor.getHTML()
    if (currentHTML !== content) {
      editor.commands.setContent(content || '<p></p>', false)
    }
  }, [content, editor])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Toolbar editor={editor} />
      <div className="editor-scroll">
        <div className="editor-content-wrapper">
          {(chapterLabel || chapterName) && (
            <div className="editor-page-header">
              {chapterLabel && <span className="editor-page-chapter-title">{chapterLabel}</span>}
              {chapterName && <span className="editor-page-chapter-name">{chapterName}</span>}
            </div>
          )}
          <div className="editor-page-body">
            <EditorContent editor={editor} />
          </div>
        </div>
      </div>
    </div>
  )
}
