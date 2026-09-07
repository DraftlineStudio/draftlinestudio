// Vite `?raw` imports resolve to the file's contents as a string.
declare module '*?raw' {
  const content: string
  export default content
}
