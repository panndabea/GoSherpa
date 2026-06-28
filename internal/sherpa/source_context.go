package sherpa

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultSourceContextRadius = 2

type SourceContext struct {
	Position Position            `json:"position"`
	Lines    []SourceContextLine `json:"lines"`
}

type SourceContextLine struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
	Target bool   `json:"target"`
}

func ReadSourceContext(root string, position Position, radius int) (SourceContext, error) {
	contexts, err := ReadSourceContexts(root, []Position{position}, radius)
	if err != nil {
		return SourceContext{}, err
	}
	if len(contexts) == 0 {
		return SourceContext{}, fmt.Errorf("source context position is empty")
	}

	return contexts[0], nil
}

func ReadSourceContexts(root string, positions []Position, radius int) ([]SourceContext, error) {
	if radius < 0 {
		return nil, fmt.Errorf("source context radius must be zero or greater")
	}

	rootPath, err := absoluteRootPath(root)
	if err != nil {
		return nil, err
	}

	reader := sourceContextReader{
		root:  rootPath,
		files: make(map[string][]string),
	}

	contexts := make([]SourceContext, 0, len(positions))
	for _, position := range positions {
		context, err := reader.read(position, radius)
		if err != nil {
			return nil, err
		}

		contexts = append(contexts, context)
	}

	return contexts, nil
}

func FormatSourceContext(context SourceContext, indent string) string {
	if len(context.Lines) == 0 {
		return ""
	}

	width := len(strconv.Itoa(context.Lines[len(context.Lines)-1].Number))
	var builder strings.Builder
	for _, line := range context.Lines {
		marker := " "
		if line.Target {
			marker = ">"
		}

		fmt.Fprintf(&builder, "%s%s %*d | %s\n", indent, marker, width, line.Number, line.Text)
	}

	return builder.String()
}

type sourceContextReader struct {
	root  string
	files map[string][]string
}

func (reader *sourceContextReader) read(position Position, radius int) (SourceContext, error) {
	filePath, relativeFile, err := sourceContextFilePath(reader.root, position.File)
	if err != nil {
		return SourceContext{}, err
	}

	lines, err := reader.readLines(filePath)
	if err != nil {
		return SourceContext{}, err
	}

	if position.Line < 1 {
		return SourceContext{}, fmt.Errorf("source context line must be greater than zero: %s:%d", relativeFile, position.Line)
	}
	if position.Line > len(lines) {
		return SourceContext{}, fmt.Errorf("source context line outside file: %s:%d", relativeFile, position.Line)
	}

	start := position.Line - radius
	if start < 1 {
		start = 1
	}
	end := position.Line + radius
	if end > len(lines) {
		end = len(lines)
	}

	context := SourceContext{
		Position: position,
	}
	context.Position.File = relativeFile
	for lineNumber := start; lineNumber <= end; lineNumber++ {
		context.Lines = append(context.Lines, SourceContextLine{
			Number: lineNumber,
			Text:   lines[lineNumber-1],
			Target: lineNumber == position.Line,
		})
	}

	return context, nil
}

func (reader *sourceContextReader) readLines(filePath string) ([]string, error) {
	if lines, ok := reader.files[filePath]; ok {
		return lines, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read source context %s: %w", filePath, err)
	}

	lines := splitSourceContextLines(string(data))
	reader.files[filePath] = lines

	return lines, nil
}

func sourceContextFilePath(root string, file string) (string, string, error) {
	if strings.TrimSpace(file) == "" {
		return "", "", fmt.Errorf("source context file is empty")
	}

	filePath := filepath.FromSlash(file)
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(root, filePath)
	}
	filePath = filepath.Clean(filePath)

	relativePath, err := filepath.Rel(root, filePath)
	if err != nil {
		return "", "", fmt.Errorf("resolve source context path %s: %w", file, err)
	}
	if relativePath == "." || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("source context file is outside repository: %s", file)
	}

	return filePath, filepath.ToSlash(relativePath), nil
}

func splitSourceContextLines(source string) []string {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")

	lines := strings.Split(source, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}
