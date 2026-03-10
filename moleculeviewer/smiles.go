package moleculeviewer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

var errUnsupportedMolVersion = errors.New("only V2000 mol blocks are supported")

type ringAnchor struct {
	atom     int
	order    int
	aromatic bool
}

// ParseSMILES parses a practical subset of SMILES suitable for small molecules.
func ParseSMILES(smiles string) (Molecule, error) {
	smiles = strings.TrimSpace(smiles)
	if smiles == "" {
		return Molecule{}, errors.New("empty SMILES")
	}

	mol := Molecule{SMILES: smiles}

	var (
		currentAtom     = -1
		branchStack     []int
		pendingOrder    = 1
		pendingAromatic bool
		rings           = map[string]ringAnchor{}
	)

	resetBond := func() {
		pendingOrder = 1
		pendingAromatic = false
	}

	for i := 0; i < len(smiles); {
		switch smiles[i] {
		case '(':
			if currentAtom < 0 {
				return Molecule{}, fmt.Errorf("branch opened before atom at position %d", i)
			}
			branchStack = append(branchStack, currentAtom)
			i++
		case ')':
			if len(branchStack) == 0 {
				return Molecule{}, fmt.Errorf("unbalanced branch close at position %d", i)
			}
			currentAtom = branchStack[len(branchStack)-1]
			branchStack = branchStack[:len(branchStack)-1]
			i++
		case '-':
			pendingOrder = 1
			pendingAromatic = false
			i++
		case '=':
			pendingOrder = 2
			pendingAromatic = false
			i++
		case '#':
			pendingOrder = 3
			pendingAromatic = false
			i++
		case ':':
			pendingOrder = 1
			pendingAromatic = true
			i++
		case '/', '\\':
			pendingOrder = 1
			pendingAromatic = false
			i++
		case '.':
			currentAtom = -1
			resetBond()
			i++
		default:
			if digit, next, ok := parseRingToken(smiles, i); ok {
				if currentAtom < 0 {
					return Molecule{}, fmt.Errorf("ring closure without anchor atom at position %d", i)
				}
				if anchor, exists := rings[digit]; exists {
					order, aromatic := resolveBond(anchor.order, pendingOrder, anchor.aromatic, pendingAromatic, mol.Atoms[anchor.atom], mol.Atoms[currentAtom])
					mol.Bonds = append(mol.Bonds, Bond{
						From:     anchor.atom,
						To:       currentAtom,
						Order:    order,
						Aromatic: aromatic,
					})
					delete(rings, digit)
				} else {
					rings[digit] = ringAnchor{
						atom:     currentAtom,
						order:    pendingOrder,
						aromatic: pendingAromatic,
					}
				}
				resetBond()
				i = next
				continue
			}

			atom, next, err := parseAtom(smiles, i)
			if err != nil {
				return Molecule{}, err
			}
			atom.Index = len(mol.Atoms)
			mol.Atoms = append(mol.Atoms, atom)

			if currentAtom >= 0 {
				order, aromatic := resolveBond(1, pendingOrder, false, pendingAromatic, mol.Atoms[currentAtom], atom)
				mol.Bonds = append(mol.Bonds, Bond{
					From:     currentAtom,
					To:       atom.Index,
					Order:    order,
					Aromatic: aromatic,
				})
			}
			currentAtom = atom.Index
			resetBond()
			i = next
		}
	}

	if len(branchStack) > 0 {
		return Molecule{}, errors.New("unbalanced branches in SMILES")
	}
	if len(rings) > 0 {
		return Molecule{}, errors.New("unclosed ring indices in SMILES")
	}

	mol.Normalize()
	return mol, nil
}

// ParseMolBlock parses a V2000 mol block and preserves existing 2D coordinates.
func ParseMolBlock(molBlock string) (Molecule, error) {
	lines := normalizeLines(molBlock)
	if len(lines) < 4 {
		return Molecule{}, errors.New("invalid mol block")
	}
	if strings.Contains(lines[3], "V3000") {
		return Molecule{}, errUnsupportedMolVersion
	}

	name := strings.TrimSpace(lines[0])
	counts := strings.Fields(lines[3])
	if len(counts) < 2 {
		return Molecule{}, errors.New("invalid counts line in mol block")
	}

	atomCount, err := strconv.Atoi(counts[0])
	if err != nil {
		return Molecule{}, fmt.Errorf("invalid atom count: %w", err)
	}
	bondCount, err := strconv.Atoi(counts[1])
	if err != nil {
		return Molecule{}, fmt.Errorf("invalid bond count: %w", err)
	}

	expected := 4 + atomCount + bondCount
	if len(lines) < expected {
		return Molecule{}, errors.New("mol block truncated before atom/bond tables completed")
	}

	mol := Molecule{Name: name}
	for i := 0; i < atomCount; i++ {
		fields := strings.Fields(lines[4+i])
		if len(fields) < 4 {
			return Molecule{}, fmt.Errorf("invalid atom line %d", i+1)
		}
		x, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return Molecule{}, fmt.Errorf("invalid atom x coordinate on line %d: %w", i+1, err)
		}
		y, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return Molecule{}, fmt.Errorf("invalid atom y coordinate on line %d: %w", i+1, err)
		}
		symbol := canonicalSymbol(fields[3])
		charge := 0
		if len(fields) >= 7 {
			charge = decodeMolCharge(fields[5])
		}
		mol.Atoms = append(mol.Atoms, Atom{
			Index:  i,
			Symbol: symbol,
			Charge: charge,
			Coords: [2]float64{x, y},
		})
	}

	for i := 0; i < bondCount; i++ {
		fields := strings.Fields(lines[4+atomCount+i])
		if len(fields) < 3 {
			return Molecule{}, fmt.Errorf("invalid bond line %d", i+1)
		}
		from, err := strconv.Atoi(fields[0])
		if err != nil {
			return Molecule{}, fmt.Errorf("invalid bond start on line %d: %w", i+1, err)
		}
		to, err := strconv.Atoi(fields[1])
		if err != nil {
			return Molecule{}, fmt.Errorf("invalid bond end on line %d: %w", i+1, err)
		}
		order, err := strconv.Atoi(fields[2])
		if err != nil {
			return Molecule{}, fmt.Errorf("invalid bond order on line %d: %w", i+1, err)
		}
		aromatic := false
		if order == 4 {
			aromatic = true
			order = 1
		}
		mol.Bonds = append(mol.Bonds, Bond{
			From:     from - 1,
			To:       to - 1,
			Order:    order,
			Aromatic: aromatic,
		})
	}

	mol.RepairConnectivityFromCoords()
	mol.Normalize()
	return mol, nil
}

// ParseSDF parses the first record from an SDF payload.
func ParseSDF(sdf string) (Molecule, error) {
	parts := strings.Split(sdf, "$$$$")
	return ParseMolBlock(parts[0])
}

// ParseMOL is an alias for ParseMolBlock.
func ParseMOL(molBlock string) (Molecule, error) {
	return ParseMolBlock(molBlock)
}

func parseAtom(smiles string, start int) (Atom, int, error) {
	if start >= len(smiles) {
		return Atom{}, start, errors.New("unexpected end of SMILES")
	}
	if smiles[start] == '[' {
		return parseBracketAtom(smiles, start)
	}
	return parseOrganicAtom(smiles, start)
}

func parseOrganicAtom(smiles string, start int) (Atom, int, error) {
	if start >= len(smiles) {
		return Atom{}, start, errors.New("unexpected end of SMILES")
	}

	switch smiles[start] {
	case 'B':
		if start+1 < len(smiles) && smiles[start+1] == 'r' {
			return Atom{Symbol: "Br"}, start + 2, nil
		}
		return Atom{Symbol: "B"}, start + 1, nil
	case 'C':
		if start+1 < len(smiles) && smiles[start+1] == 'l' {
			return Atom{Symbol: "Cl"}, start + 2, nil
		}
		return Atom{Symbol: "C"}, start + 1, nil
	case 'N':
		return Atom{Symbol: "N"}, start + 1, nil
	case 'O':
		return Atom{Symbol: "O"}, start + 1, nil
	case 'P':
		return Atom{Symbol: "P"}, start + 1, nil
	case 'S':
		return Atom{Symbol: "S"}, start + 1, nil
	case 'F':
		return Atom{Symbol: "F"}, start + 1, nil
	case 'I':
		return Atom{Symbol: "I"}, start + 1, nil
	case 'c':
		return Atom{Symbol: "C", Aromatic: true}, start + 1, nil
	case 'n':
		return Atom{Symbol: "N", Aromatic: true}, start + 1, nil
	case 'o':
		return Atom{Symbol: "O", Aromatic: true}, start + 1, nil
	case 'p':
		return Atom{Symbol: "P", Aromatic: true}, start + 1, nil
	case 's':
		return Atom{Symbol: "S", Aromatic: true}, start + 1, nil
	case 'b':
		return Atom{Symbol: "B", Aromatic: true}, start + 1, nil
	case 'H':
		return Atom{Symbol: "H"}, start + 1, nil
	default:
		return Atom{}, start, fmt.Errorf("unsupported atom token %q at position %d", smiles[start:start+1], start)
	}
}

func parseBracketAtom(smiles string, start int) (Atom, int, error) {
	end := strings.IndexByte(smiles[start:], ']')
	if end < 0 {
		return Atom{}, start, errors.New("unclosed bracket atom")
	}
	end += start
	body := smiles[start+1 : end]
	if body == "" {
		return Atom{}, start, errors.New("empty bracket atom")
	}

	j := 0
	for j < len(body) && unicode.IsDigit(rune(body[j])) {
		j++
	}

	if j >= len(body) {
		return Atom{}, start, errors.New("missing atom symbol in bracket atom")
	}

	aromatic := unicode.IsLower(rune(body[j]))
	symbol, consumed := parseSymbolToken(body[j:])
	if consumed == 0 {
		return Atom{}, start, errors.New("invalid bracket atom symbol")
	}
	j += consumed

	atom := Atom{
		Symbol:   canonicalSymbol(symbol),
		Aromatic: aromatic,
	}

	for j < len(body) {
		switch body[j] {
		case '@':
			j++
			if j < len(body) && body[j] == '@' {
				j++
			}
		case 'H':
			j++
			hydrogens := 1
			if j < len(body) && unicode.IsDigit(rune(body[j])) {
				value, next := parseDigits(body, j)
				hydrogens = value
				j = next
			}
			atom.Hydrogens = hydrogens
		case '+', '-':
			sign := 1
			if body[j] == '-' {
				sign = -1
			}
			j++
			magnitude := 1
			if j < len(body) && unicode.IsDigit(rune(body[j])) {
				value, next := parseDigits(body, j)
				magnitude = value
				j = next
			} else {
				magnitude = 1
				for j < len(body) && (body[j] == '+' || body[j] == '-') {
					if sign == 1 && body[j] == '+' {
						magnitude++
						j++
						continue
					}
					if sign == -1 && body[j] == '-' {
						magnitude++
						j++
						continue
					}
					break
				}
			}
			atom.Charge = sign * magnitude
		case ':':
			j++
			for j < len(body) && unicode.IsDigit(rune(body[j])) {
				j++
			}
		default:
			j++
		}
	}

	return atom, end + 1, nil
}

func parseSymbolToken(body string) (string, int) {
	if body == "" {
		return "", 0
	}
	if len(body) >= 2 {
		token := body[:2]
		switch token {
		case "Cl", "Br", "Si", "Na", "Li", "Mg", "Ca", "Zn", "Fe", "Al", "cl", "br", "si", "na", "li", "mg", "ca", "zn", "fe", "al":
			return token, 2
		}
	}
	if unicode.IsLetter(rune(body[0])) {
		return body[:1], 1
	}
	return "", 0
}

func parseDigits(s string, start int) (int, int) {
	end := start
	for end < len(s) && unicode.IsDigit(rune(s[end])) {
		end++
	}
	value, _ := strconv.Atoi(s[start:end])
	return value, end
}

func parseRingToken(smiles string, start int) (string, int, bool) {
	if start >= len(smiles) {
		return "", start, false
	}
	if unicode.IsDigit(rune(smiles[start])) {
		return smiles[start : start+1], start + 1, true
	}
	if smiles[start] == '%' && start+2 < len(smiles) &&
		unicode.IsDigit(rune(smiles[start+1])) &&
		unicode.IsDigit(rune(smiles[start+2])) {
		return smiles[start+1 : start+3], start + 3, true
	}
	return "", start, false
}

func resolveBond(openOrder, pendingOrder int, openAromatic, pendingAromatic bool, a, b Atom) (int, bool) {
	if pendingAromatic || openAromatic || (pendingOrder == 1 && openOrder == 1 && a.Aromatic && b.Aromatic) {
		return 1, true
	}
	if pendingOrder > 1 {
		return pendingOrder, false
	}
	if openOrder > 1 {
		return openOrder, false
	}
	return 1, false
}

func normalizeLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func decodeMolCharge(code string) int {
	switch strings.TrimSpace(code) {
	case "0", "":
		return 0
	case "1":
		return 3
	case "2":
		return 2
	case "3":
		return 1
	case "5":
		return -1
	case "6":
		return -2
	case "7":
		return -3
	default:
		return 0
	}
}
