package jsonc

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// ParseObject decodes a JSON object while accepting line and block comments.
func ParseObject(content []byte) (map[string]any, error) {
	if strings.TrimSpace(string(content)) == "" {
		return map[string]any{}, nil
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(StripComments(string(content))))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("settings file contains trailing JSON data")
		}
		return nil, err
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("settings file is not a JSON object")
	}
	return root, nil
}

func StripComments(content string) string {
	stripper := commentStripper{content: content}
	for stripper.index < len(content) {
		stripper.writeNext()
	}
	return stripper.builder.String()
}

type commentStripper struct {
	content  string
	builder  strings.Builder
	inString bool
	escaped  bool
	index    int
}

func (s *commentStripper) writeNext() {
	ch := s.content[s.index]
	if s.writeEscaped(ch) || s.writeStringContent(ch) || s.writeStringStart(ch) || s.skipComment() {
		return
	}
	s.builder.WriteByte(ch)
	s.index++
}

func (s *commentStripper) writeEscaped(ch byte) bool {
	if !s.escaped {
		return false
	}
	s.builder.WriteByte(ch)
	s.escaped = false
	s.index++
	return true
}

func (s *commentStripper) writeStringContent(ch byte) bool {
	if !s.inString {
		return false
	}
	s.builder.WriteByte(ch)
	s.escaped = ch == '\\'
	if ch == '"' {
		s.inString = false
	}
	s.index++
	return true
}

func (s *commentStripper) writeStringStart(ch byte) bool {
	if ch != '"' {
		return false
	}
	s.inString = true
	s.builder.WriteByte(ch)
	s.index++
	return true
}

func (s *commentStripper) skipComment() bool {
	if s.content[s.index] != '/' || s.index+1 >= len(s.content) {
		return false
	}
	switch s.content[s.index+1] {
	case '/':
		s.skipLineComment()
		return true
	case '*':
		s.skipBlockComment()
		return true
	default:
		return false
	}
}

func (s *commentStripper) skipLineComment() {
	s.index += 2
	for s.index < len(s.content) && !isLineBreak(s.content[s.index]) {
		s.index++
	}
	if s.index < len(s.content) {
		s.builder.WriteByte(s.content[s.index])
		s.index++
	}
}

func (s *commentStripper) skipBlockComment() {
	s.index += 2
	for s.index+1 < len(s.content) && !s.atBlockCommentEnd() {
		if isLineBreak(s.content[s.index]) {
			s.builder.WriteByte(s.content[s.index])
		}
		s.index++
	}
	if s.index+1 < len(s.content) {
		s.index += 2
	}
}

func (s *commentStripper) atBlockCommentEnd() bool {
	return s.content[s.index] == '*' && s.content[s.index+1] == '/'
}

func isLineBreak(ch byte) bool {
	return ch == '\n' || ch == '\r'
}
