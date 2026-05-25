import { marked } from 'marked'
import DOMPurify from 'dompurify'

export function useMarkdown(body: string): string {
  return DOMPurify.sanitize(marked.parse(body) as string)
}
