declare module 'typo-js' {
  export default class Typo {
    constructor(
      dictionary: string,
      affData?: string | null,
      dicData?: string | null,
      settings?: {
        dictionaryPath?: string
        asyncLoad?: boolean
        loadedCallback?: () => void
      }
    )
    check(word: string): boolean
    suggest(word: string, limit?: number): string[]
    loaded: boolean
  }
}
