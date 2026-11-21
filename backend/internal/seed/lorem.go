package seed

import "strings"

// Placeholder text for the demo project. It is built from a fixed word list and
// the position of the ticket or comment, with no randomness, so every install
// shows the same demo data.

var loremWords = strings.Fields(`lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor
incididunt ut labore et dolore magna aliqua enim ad minim veniam quis nostrud exercitation ullamco laboris nisi
aliquip ex ea commodo consequat duis aute irure in reprehenderit voluptate velit esse cillum fugiat nulla
pariatur excepteur sint occaecat cupidatat non proident sunt culpa qui officia deserunt mollit anim id est
laborum`)

// loremSentence returns one sentence of 7 to 14 words. The same seed always gives
// the same sentence.
func loremSentence(seed int) string {
	n := 7 + (seed*5)%8
	words := make([]string, n)
	for i := range words {
		words[i] = loremWords[(seed*7+i*3+i*i)%len(loremWords)]
	}
	s := strings.Join(words, " ")
	return strings.ToUpper(s[:1]) + s[1:] + "."
}

// loremParagraph returns count sentences, starting from seed.
func loremParagraph(seed, count int) string {
	sentences := make([]string, count)
	for i := range sentences {
		sentences[i] = loremSentence(seed*11 + i*17 + 1)
	}
	return strings.Join(sentences, " ")
}

// loremDescription is a ticket description in markdown: it always opens with the
// classic first line, then a paragraph, and stories and bugs get a short list.
func loremDescription(index int, typ string) string {
	parts := []string{
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. " + loremParagraph(index+1, 2+index%2),
		loremParagraph(index+40, 3),
	}
	switch typ {
	case "story":
		parts = append(parts, "**Acceptance criteria**\n\n- "+loremSentence(index+70)+"\n- "+loremSentence(index+71)+"\n- "+loremSentence(index+72))
	case "bug":
		parts = append(parts, "**Steps to reproduce**\n\n1. "+loremSentence(index+80)+"\n2. "+loremSentence(index+81)+"\n3. "+loremSentence(index+82))
	}
	return strings.Join(parts, "\n\n")
}

// loremComment is the text of one comment. Every few comments is a longer one.
func loremComment(issueIndex, n int) string {
	sentences := 1 + (issueIndex+n)%3
	return loremParagraph(issueIndex*31+n*13+5, sentences)
}
