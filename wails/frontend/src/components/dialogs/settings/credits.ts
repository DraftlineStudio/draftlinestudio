// Open-source projects Draftline is built from, shown in Settings → About.
// Websites and license links are the projects' own canonical addresses.

export interface Credit {
  id: string
  name: string
  role: string
  license: string
  website: string
  source?: string
  licenseUrl: string
}

export interface CreditGroup {
  label: string
  credits: Credit[]
}

export const CREDIT_GROUPS: CreditGroup[] = [
  {
    label: 'Application & Interface',
    credits: [
      {
        id: 'wails',
        name: 'Wails',
        role: 'The desktop application framework that joins the Go backend to the interface.',
        license: 'MIT',
        website: 'https://wails.io',
        source: 'https://github.com/wailsapp/wails',
        licenseUrl: 'https://github.com/wailsapp/wails/blob/master/LICENSE',
      },
      {
        id: 'react',
        name: 'React',
        role: 'The user interface library everything on screen is built with.',
        license: 'MIT',
        website: 'https://react.dev',
        source: 'https://github.com/facebook/react',
        licenseUrl: 'https://github.com/facebook/react/blob/main/LICENSE',
      },
      {
        id: 'tiptap',
        name: 'Tiptap',
        role: 'The rich-text editor framework behind the manuscript editor.',
        license: 'MIT',
        website: 'https://tiptap.dev',
        source: 'https://github.com/ueberdosis/tiptap',
        licenseUrl: 'https://github.com/ueberdosis/tiptap/blob/main/LICENSE.md',
      },
      {
        id: 'prosemirror',
        name: 'ProseMirror',
        role: 'The editing engine underneath Tiptap: the document model, cursor, and undo history.',
        license: 'MIT',
        website: 'https://prosemirror.net',
        source: 'https://github.com/ProseMirror/prosemirror',
        licenseUrl: 'https://github.com/ProseMirror/prosemirror/blob/master/LICENSE',
      },
      {
        id: 'zustand',
        name: 'Zustand',
        role: 'State management for the interface.',
        license: 'MIT',
        website: 'https://github.com/pmndrs/zustand',
        licenseUrl: 'https://github.com/pmndrs/zustand/blob/main/LICENSE',
      },
      {
        id: 'dnd-kit',
        name: 'dnd kit',
        role: 'Drag and drop, used for reordering chapters.',
        license: 'MIT',
        website: 'https://dndkit.com',
        source: 'https://github.com/clauderic/dnd-kit',
        licenseUrl: 'https://github.com/clauderic/dnd-kit/blob/master/LICENSE',
      },
      {
        id: 'go',
        name: 'Go',
        role: 'The programming language the backend is written in, with its golang.org/x libraries.',
        license: 'BSD-3-Clause',
        website: 'https://go.dev',
        licenseUrl: 'https://go.dev/LICENSE',
      },
      {
        id: 'go-keyring',
        name: 'go-keyring',
        role: 'Stores API keys in the operating system keychain instead of a file.',
        license: 'MIT',
        website: 'https://github.com/zalando/go-keyring',
        licenseUrl: 'https://github.com/zalando/go-keyring/blob/master/LICENSE',
      },
      {
        id: 'fpdf',
        name: 'go-pdf/fpdf',
        role: 'PDF generation for manuscript export.',
        license: 'MIT',
        website: 'https://codeberg.org/go-pdf/fpdf',
        licenseUrl: 'https://codeberg.org/go-pdf/fpdf/src/branch/main/LICENSE',
      },
      {
        id: 'purego',
        name: 'purego',
        role: 'Lets the Go backend call native libraries, used to load the speech engine.',
        license: 'Apache-2.0',
        website: 'https://github.com/ebitengine/purego',
        licenseUrl: 'https://github.com/ebitengine/purego/blob/main/LICENSE',
      },
    ],
  },
  {
    label: 'Language Analysis & Speech',
    credits: [
      {
        id: 'typo-js',
        name: 'Typo.js',
        role: 'The Hunspell-compatible spell checker behind the red underlines.',
        license: 'BSD-3-Clause',
        website: 'https://github.com/cfinke/Typo.js',
        licenseUrl: 'https://github.com/cfinke/Typo.js/blob/master/license.txt',
      },
      {
        id: 'prose',
        name: 'prose',
        role: 'Natural-language analysis in Go: sentences, parts of speech, and names for story analysis.',
        license: 'MIT',
        website: 'https://github.com/jdkato/prose',
        licenseUrl: 'https://github.com/jdkato/prose/blob/master/LICENSE',
      },
      {
        id: 'sherpa-onnx',
        name: 'sherpa-onnx',
        role: 'The speech synthesis runtime behind Read Aloud, downloaded when you enable it.',
        license: 'Apache-2.0',
        website: 'https://github.com/k2-fsa/sherpa-onnx',
        licenseUrl: 'https://github.com/k2-fsa/sherpa-onnx/blob/master/LICENSE',
      },
      {
        id: 'onnxruntime',
        name: 'ONNX Runtime',
        role: 'Runs the Read Aloud voice model on your machine.',
        license: 'MIT',
        website: 'https://onnxruntime.ai',
        source: 'https://github.com/microsoft/onnxruntime',
        licenseUrl: 'https://github.com/microsoft/onnxruntime/blob/main/LICENSE',
      },
      {
        id: 'kokoro',
        name: 'Kokoro',
        role: 'The Read Aloud voice model.',
        license: 'Apache-2.0',
        website: 'https://huggingface.co/hexgrad/Kokoro-82M',
        licenseUrl: 'https://www.apache.org/licenses/LICENSE-2.0',
      },
      {
        id: 'espeak-ng',
        name: 'eSpeak NG',
        role: 'Pronunciation data the voice model uses to say unfamiliar words.',
        license: 'GPL-3.0',
        website: 'https://github.com/espeak-ng/espeak-ng',
        licenseUrl: 'https://github.com/espeak-ng/espeak-ng/blob/master/COPYING',
      },
    ],
  },
  {
    label: 'Dictionaries & Data',
    credits: [
      {
        id: 'scowl',
        name: 'SCOWL',
        role: 'The word lists the American, British, Canadian, and Australian spelling dictionaries are built from.',
        license: 'MIT-style',
        website: 'http://wordlist.aspell.net',
        licenseUrl: 'http://wordlist.aspell.net/scowl-readme/',
      },
      {
        id: 'wiktionary',
        name: 'Wiktionary',
        role: 'Regional vocabulary notes are derived from Wiktionary, extracted with wiktextract (kaikki.org).',
        license: 'CC BY-SA',
        website: 'https://www.wiktionary.org',
        source: 'https://kaikki.org',
        licenseUrl: 'https://creativecommons.org/licenses/by-sa/4.0/',
      },
    ],
  },
  {
    label: 'Fonts',
    credits: [
      {
        id: 'ibm-plex',
        name: 'IBM Plex Sans',
        role: 'The interface typeface.',
        license: 'OFL-1.1',
        website: 'https://github.com/IBM/plex',
        licenseUrl: 'https://github.com/IBM/plex/blob/master/LICENSE.txt',
      },
      {
        id: 'merriweather',
        name: 'Merriweather',
        role: 'A manuscript typeface option in the editor.',
        license: 'OFL-1.1',
        website: 'https://github.com/SorkinType/Merriweather',
        licenseUrl: 'https://github.com/SorkinType/Merriweather/blob/master/OFL.txt',
      },
    ],
  },
]
