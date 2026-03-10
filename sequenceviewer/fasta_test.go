package sequenceviewer

import "testing"

func TestParseFASTASingleRecord(t *testing.T) {
	records, err := ParseFASTA(">seq1 Example\nATGC\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].ID != "seq1" || records[0].Description != "Example" {
		t.Fatalf("unexpected record header %+v", records[0])
	}
	if records[0].Sequence != "ATGC" {
		t.Fatalf("unexpected record sequence %q", records[0].Sequence)
	}
}

func TestParseFASTAMultiRecordMultiline(t *testing.T) {
	data := ">seq1\nATG\nCGA\n>seq2 second\nMKWV\nTFI\n"
	records, err := ParseFASTA(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Sequence != "ATGCGA" {
		t.Fatalf("unexpected first sequence %q", records[0].Sequence)
	}
	if records[1].Description != "second" {
		t.Fatalf("unexpected second description %q", records[1].Description)
	}
}

func TestParseFASTACommentsAndBlankLines(t *testing.T) {
	data := "; comment\n\n>seq1\nATGC\n; trailing comment\n"
	records, err := ParseFASTA(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
}

func TestParseFASTAEmpty(t *testing.T) {
	records, err := ParseFASTA("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestParseFASTARequiresHeader(t *testing.T) {
	if _, err := ParseFASTA("ATGC"); err == nil {
		t.Fatal("expected error for FASTA content without a header")
	}
}
