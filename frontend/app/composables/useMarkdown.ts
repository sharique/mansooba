import { marked } from 'marked'

export function useMarkdown(body: string): string {
  return marked.parse(body) as string
}
