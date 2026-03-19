export type Section = 'copyright' | 'front_matter' | 'body' | 'back_matter'

export type ProjectType = 'book' | 'universe'

export interface RecentProject {
  type: ProjectType
  path: string
  name: string
  lastOpened: string  // ISO date string
  stats: {
    books?: number
    chapters: number
    words: number
  }
}

export type CharacterRole = 'protagonist' | 'antagonist' | 'supporting' | 'minor' | 'other'

export interface Character {
  id: string
  name: string
  role: CharacterRole | string
  description: string
  appearance: string
  personality: string
  motivation: string
  notes: string
}

export interface StoryBible {
  characters: Character[]
  plot_notes: string
  timeline: string
}

export interface Metadata {
  title: string
  author: string
  isbn: string
  publisher: string
  created: string
  modified: string
}

export interface ChapterItem {
  title: string
  type: string
  content: string
}

export interface BookData {
  version: string
  metadata: Metadata
  copyright: string
  front_matter: ChapterItem[]
  body: ChapterItem[]
  back_matter: ChapterItem[]
  file_path?: string
  story_bible?: StoryBible
}

export interface SaveResult {
  success: boolean
  file_path: string
  error?: string
}

export const FRONT_MATTER_TYPES = [
  'Author\'s Note',
  'Foreword',
  'Preface',
  'Introduction',
  'Prologue',
  'Dedication',
]

export const BODY_TYPES = [
  'Chapter',
  'Interlude',
  'Part',
  'Scene',
]

export const BACK_MATTER_TYPES = [
  'Epilogue',
  'Afterword',
  'Appendix',
  'About the Author',
  'Acknowledgments',
  'Glossary',
]

export const EDITOR_FONTS = [
  'Merriweather',
  'Georgia',
  'Times New Roman',
  'Garamond',
  'IBM Plex Sans',
  'Arial',
]

export const EDITOR_FONT_SIZES = ['12', '13', '14', '15', '16', '18', '20', '24']
