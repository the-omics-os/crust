package sequenceviewer

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ParseFASTA parses FASTA records from a string.
func ParseFASTA(data string) ([]FASTARecord, error) {
	return ParseFASTAReader(strings.NewReader(data))
}

// ParseFASTAReader parses FASTA records from an io.Reader.
func ParseFASTAReader(r io.Reader) ([]FASTARecord, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var records []FASTARecord
	var current *FASTARecord

	flush := func() {
		if current == nil {
			return
		}
		current.Sequence = NormalizeSequence(current.Sequence)
		records = append(records, *current)
		current = nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, ">") {
			flush()
			header := strings.TrimSpace(strings.TrimPrefix(line, ">"))
			id, desc := splitFASTAHeader(header)
			current = &FASTARecord{ID: id, Description: desc}
			continue
		}
		if current == nil {
			return nil, fmt.Errorf("fasta sequence line encountered before header")
		}
		current.Sequence += line
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	flush()
	return records, nil
}

func splitFASTAHeader(header string) (id, description string) {
	if header == "" {
		return "", ""
	}
	parts := strings.Fields(header)
	if len(parts) == 0 {
		return "", ""
	}
	id = parts[0]
	if len(parts) > 1 {
		description = strings.Join(parts[1:], " ")
	}
	return id, description
}
