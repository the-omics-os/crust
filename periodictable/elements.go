package periodictable

import (
	"strconv"
	"strings"
)

const (
	categoryAlkaliMetal     = "alkali-metal"
	categoryAlkalineEarth   = "alkaline-earth"
	categoryTransitionMetal = "transition-metal"
	categoryPostTransition  = "post-transition-metal"
	categoryMetalloid       = "metalloid"
	categoryNonmetal        = "nonmetal"
	categoryHalogen         = "halogen"
	categoryNobleGas        = "noble-gas"
	categoryLanthanide      = "lanthanide"
	categoryActinide        = "actinide"
)

// Element describes a single chemical element in the periodic table.
type Element struct {
	Number            int
	Symbol            string
	Name              string
	AtomicMass        float64
	Group             int
	Period            int
	Category          string
	Electronegativity float64
	ElectronConfig    string
	VdwRadius         float64
	CovalentRadius    float64

	row int
	col int
}

var (
	allElements []Element
	numberIndex map[int]int
	symbolIndex map[string]int
	displayGrid [10][19]int
)

func init() {
	lines := strings.Split(strings.TrimSpace(rawElementData), "\n")
	allElements = make([]Element, 0, len(lines))
	numberIndex = make(map[int]int, len(lines))
	symbolIndex = make(map[string]int, len(lines))

	for _, line := range lines {
		fields := strings.Split(line, "|")
		if len(fields) != 13 {
			panic("periodictable: invalid element record")
		}

		el := Element{
			Number:            mustInt(fields[0]),
			Symbol:            fields[1],
			Name:              fields[2],
			AtomicMass:        mustFloat(fields[3]),
			Group:             mustInt(fields[4]),
			Period:            mustInt(fields[5]),
			Category:          fields[6],
			Electronegativity: mustFloat(fields[7]),
			ElectronConfig:    fields[8],
			VdwRadius:         mustFloat(fields[9]),
			CovalentRadius:    mustFloat(fields[10]),
			row:               mustInt(fields[11]),
			col:               mustInt(fields[12]),
		}

		allElements = append(allElements, el)
		idx := len(allElements) - 1
		numberIndex[el.Number] = idx
		symbolIndex[el.Symbol] = idx
		displayGrid[el.row][el.col] = el.Number
	}

	if len(allElements) != 118 {
		panic("periodictable: expected 118 elements")
	}
}

func elementByNumber(number int) (Element, bool) {
	idx, ok := numberIndex[number]
	if !ok {
		return Element{}, false
	}
	return allElements[idx], true
}

func elementBySymbol(symbol string) (Element, bool) {
	idx, ok := symbolIndex[canonicalSymbol(symbol)]
	if !ok {
		return Element{}, false
	}
	return allElements[idx], true
}

func elementAt(row, col int) (Element, bool) {
	if row < 1 || row > 9 || col < 1 || col > 18 {
		return Element{}, false
	}
	number := displayGrid[row][col]
	if number == 0 {
		return Element{}, false
	}
	return elementByNumber(number)
}

func elementPosition(number int) (row, col int, ok bool) {
	el, found := elementByNumber(number)
	if !found {
		return 0, 0, false
	}
	return el.row, el.col, true
}

func mustInt(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		panic(err)
	}
	return n
}

func mustFloat(value string) float64 {
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic(err)
	}
	return n
}

// Radii values are stored in angstroms using the best available hardcoded
// dataset for this package. When a source omits a radius, the value is 0.
const rawElementData = `
1|H|Hydrogen|1.008000|1|1|nonmetal|2.20|1s1|0.79|0.32|1|1
2|He|Helium|4.002602|18|1|noble-gas|0.00|1s2|0.00|0.00|1|18
3|Li|Lithium|6.940000|1|2|alkali-metal|0.98|[He] 2s1|1.55|1.63|2|1
4|Be|Beryllium|9.012183|2|2|alkaline-earth|1.57|[He] 2s2|1.12|0.90|2|2
5|B|Boron|10.810000|13|2|metalloid|2.04|[He] 2s2 2p1|0.98|0.82|2|13
6|C|Carbon|12.011000|14|2|nonmetal|2.55|[He] 2s2 2p2|0.91|0.77|2|14
7|N|Nitrogen|14.007000|15|2|nonmetal|3.04|[He] 2s2 2p3|0.92|0.75|2|15
8|O|Oxygen|15.999000|16|2|nonmetal|3.44|[He] 2s2 2p4|0.00|0.73|2|16
9|F|Fluorine|18.998403|17|2|nonmetal|3.98|[He] 2s2 2p5|0.00|0.72|2|17
10|Ne|Neon|20.179760|18|2|noble-gas|0.00|[He] 2s2 2p6|0.00|0.71|2|18
11|Na|Sodium|22.989769|1|3|alkali-metal|0.93|[Ne] 3s1|1.90|1.54|3|1
12|Mg|Magnesium|24.305000|2|3|alkaline-earth|1.31|[Ne] 3s2|1.60|1.36|3|2
13|Al|Aluminium|26.981539|13|3|post-transition-metal|1.61|[Ne] 3s2 3p1|0.00|0.00|3|13
14|Si|Silicon|28.085000|14|3|metalloid|1.90|[Ne] 3s2 3p2|1.32|1.11|3|14
15|P|Phosphorus|30.973762|15|3|nonmetal|2.19|[Ne] 3s2 3p3|1.28|1.06|3|15
16|S|Sulfur|32.060000|16|3|nonmetal|2.58|[Ne] 3s2 3p4|1.27|1.02|3|16
17|Cl|Chlorine|35.450000|17|3|nonmetal|3.16|[Ne] 3s2 3p5|0.00|0.99|3|17
18|Ar|Argon|39.948100|18|3|noble-gas|0.00|[Ne] 3s2 3p6|0.02|0.98|3|18
19|K|Potassium|39.098310|1|4|alkali-metal|0.82|[Ar] 4s1|2.35|2.03|4|1
20|Ca|Calcium|40.078400|2|4|alkaline-earth|1.00|[Ar] 4s2|1.97|1.74|4|2
21|Sc|Scandium|44.955908|3|4|transition-metal|1.36|[Ar] 3d1 4s2|1.62|1.44|4|3
22|Ti|Titanium|47.867100|4|4|transition-metal|1.54|[Ar] 3d2 4s2|1.47|1.32|4|4
23|V|Vanadium|50.941510|5|4|transition-metal|1.63|[Ar] 3d3 4s2|1.34|1.22|4|5
24|Cr|Chromium|51.996160|6|4|transition-metal|1.66|[Ar] 3d5 4s1|1.30|1.18|4|6
25|Mn|Manganese|54.938044|7|4|transition-metal|1.55|[Ar] 3d5 4s2|1.35|1.17|4|7
26|Fe|Iron|55.845200|8|4|transition-metal|1.83|[Ar] 3d6 4s2|1.26|1.17|4|8
27|Co|Cobalt|58.933194|9|4|transition-metal|1.88|[Ar] 3d7 4s2|1.25|1.16|4|9
28|Ni|Nickel|58.693440|10|4|transition-metal|1.91|[Ar] 3d8 4s2|1.24|1.15|4|10
29|Cu|Copper|63.546300|11|4|transition-metal|1.90|[Ar] 3d10 4s1|1.28|1.17|4|11
30|Zn|Zinc|65.382000|12|4|transition-metal|1.65|[Ar] 3d10 4s2|1.38|1.25|4|12
31|Ga|Gallium|69.723100|13|4|post-transition-metal|1.81|[Ar] 3d10 4s2 4p1|1.41|1.26|4|13
32|Ge|Germanium|72.630800|14|4|metalloid|2.01|[Ar] 3d10 4s2 4p2|1.37|1.22|4|14
33|As|Arsenic|74.921596|15|4|metalloid|2.18|[Ar] 3d10 4s2 4p3|1.39|1.20|4|15
34|Se|Selenium|78.971800|16|4|nonmetal|2.55|[Ar] 3d10 4s2 4p4|1.40|1.16|4|16
35|Br|Bromine|79.904000|17|4|nonmetal|2.96|[Ar] 3d10 4s2 4p5|0.00|1.14|4|17
36|Kr|Krypton|83.798200|18|4|noble-gas|3.00|[Ar] 3d10 4s2 4p6|0.00|1.12|4|18
37|Rb|Rubidium|85.467830|1|5|alkali-metal|0.82|[Kr] 5s1|2.48|2.16|5|1
38|Sr|Strontium|87.621000|2|5|alkaline-earth|0.95|[Kr] 5s2|2.15|1.91|5|2
39|Y|Yttrium|88.905842|3|5|transition-metal|1.22|[Kr] 4d1 5s2|1.78|1.62|5|3
40|Zr|Zirconium|91.224200|4|5|transition-metal|1.33|[Kr] 4d2 5s2|1.60|1.45|5|4
41|Nb|Niobium|92.906372|5|5|transition-metal|1.60|[Kr] 4d4 5s1|1.46|1.34|5|5
42|Mo|Molybdenum|95.951000|6|5|transition-metal|2.16|[Kr] 4d5 5s1|1.39|1.30|5|6
43|Tc|Technetium|98.000000|7|5|transition-metal|1.90|[Kr] 4d5 5s2|1.36|1.27|5|7
44|Ru|Ruthenium|101.072000|8|5|transition-metal|2.20|[Kr] 4d7 5s1|1.34|1.25|5|8
45|Rh|Rhodium|102.905502|9|5|transition-metal|2.28|[Kr] 4d8 5s1|1.34|1.25|5|9
46|Pd|Palladium|106.421000|10|5|transition-metal|2.20|[Kr] 4d10|1.37|1.28|5|10
47|Ag|Silver|107.868220|11|5|transition-metal|1.93|[Kr] 4d10 5s1|1.44|1.34|5|11
48|Cd|Cadmium|112.414400|12|5|transition-metal|1.69|[Kr] 4d10 5s2|1.54|1.48|5|12
49|In|Indium|114.818100|13|5|post-transition-metal|1.78|[Kr] 4d10 5s2 5p1|1.66|1.44|5|13
50|Sn|Tin|118.710700|14|5|post-transition-metal|1.96|[Kr] 4d10 5s2 5p2|1.62|1.41|5|14
51|Sb|Antimony|121.760100|15|5|metalloid|2.05|[Kr] 4d10 5s2 5p3|1.59|1.40|5|15
52|Te|Tellurium|127.603000|16|5|metalloid|2.10|[Kr] 4d10 5s2 5p4|1.60|1.36|5|16
53|I|Iodine|126.904473|17|5|nonmetal|2.66|[Kr] 4d10 5s2 5p5|0.00|1.33|5|17
54|Xe|Xenon|131.293600|18|5|noble-gas|2.60|[Kr] 4d10 5s2 5p6|0.00|1.31|5|18
55|Cs|Cesium|132.905452|1|6|alkali-metal|0.79|[Xe] 6s1|2.67|2.35|6|1
56|Ba|Barium|137.327700|2|6|alkaline-earth|0.89|[Xe] 6s2|2.22|1.98|6|2
57|La|Lanthanum|138.905477|0|6|lanthanide|1.10|[Xe] 5d1 6s2|1.87|1.69|8|3
58|Ce|Cerium|140.116100|0|6|lanthanide|1.12|[Xe] 4f1 5d1 6s2|1.81|1.65|8|4
59|Pr|Praseodymium|140.907662|0|6|lanthanide|1.13|[Xe] 4f3 6s2|1.82|1.65|8|5
60|Nd|Neodymium|144.242300|0|6|lanthanide|1.14|[Xe] 4f4 6s2|1.82|1.84|8|6
61|Pm|Promethium|145.000000|0|6|lanthanide|1.13|[Xe] 4f5 6s2|0.00|1.63|8|7
62|Sm|Samarium|150.362000|0|6|lanthanide|1.17|[Xe] 4f6 6s2|1.81|1.62|8|8
63|Eu|Europium|151.964100|0|6|lanthanide|1.20|[Xe] 4f7 6s2|1.99|1.85|8|9
64|Gd|Gadolinium|157.253000|0|6|lanthanide|1.20|[Xe] 4f7 5d1 6s2|1.79|1.61|8|10
65|Tb|Terbium|158.925352|0|6|lanthanide|1.10|[Xe] 4f9 6s2|1.80|1.59|8|11
66|Dy|Dysprosium|162.500100|0|6|lanthanide|1.22|[Xe] 4f10 6s2|1.80|1.59|8|12
67|Ho|Holmium|164.930332|0|6|lanthanide|1.23|[Xe] 4f11 6s2|1.79|1.58|8|13
68|Er|Erbium|167.259300|0|6|lanthanide|1.24|[Xe] 4f12 6s2|1.78|1.57|8|14
69|Tm|Thulium|168.934222|0|6|lanthanide|1.25|[Xe] 4f13 6s2|1.77|1.56|8|15
70|Yb|Ytterbium|173.045100|0|6|lanthanide|1.10|[Xe] 4f14 6s2|1.94|0.00|8|16
71|Lu|Lutetium|174.966810|0|6|lanthanide|1.27|[Xe] 4f14 5d1 6s2|1.75|1.56|8|17
72|Hf|Hafnium|178.492000|4|6|transition-metal|1.30|[Xe] 4f14 5d2 6s2|1.67|1.44|6|4
73|Ta|Tantalum|180.947882|5|6|transition-metal|1.50|[Xe] 4f14 5d3 6s2|1.49|1.34|6|5
74|W|Tungsten|183.841000|6|6|transition-metal|2.36|[Xe] 4f14 5d4 6s2|1.41|1.30|6|6
75|Re|Rhenium|186.207100|7|6|transition-metal|1.90|[Xe] 4f14 5d5 6s2|1.37|1.28|6|7
76|Os|Osmium|190.233000|8|6|transition-metal|2.20|[Xe] 4f14 5d6 6s2|1.35|1.26|6|8
77|Ir|Iridium|192.217300|9|6|transition-metal|2.20|[Xe] 4f14 5d7 6s2|1.36|1.27|6|9
78|Pt|Platinum|195.084900|10|6|transition-metal|2.28|[Xe] 4f14 5d9 6s1|1.39|1.30|6|10
79|Au|Gold|196.966569|11|6|transition-metal|2.54|[Xe] 4f14 5d10 6s1|1.46|1.34|6|11
80|Hg|Mercury|200.592300|12|6|transition-metal|2.00|[Xe] 4f14 5d10 6s2|1.57|1.49|6|12
81|Tl|Thallium|204.380000|13|6|post-transition-metal|1.62|[Xe] 4f14 5d10 6s2 6p1|1.71|1.48|6|13
82|Pb|Lead|207.210000|14|6|post-transition-metal|1.87|[Xe] 4f14 5d10 6s2 6p2|1.75|1.47|6|14
83|Bi|Bismuth|208.980401|15|6|post-transition-metal|2.02|[Xe] 4f14 5d10 6s2 6p3|1.70|1.46|6|15
84|Po|Polonium|209.000000|16|6|post-transition-metal|2.00|[Xe] 4f14 5d10 6s2 6p4|1.76|1.46|6|16
85|At|Astatine|210.000000|17|6|halogen|2.20|[Xe] 4f14 5d10 6s2 6p5|0.00|1.45|6|17
86|Rn|Radon|222.000000|18|6|noble-gas|2.20|[Xe] 4f14 5d10 6s2 6p6|0.00|0.00|6|18
87|Fr|Francium|223.000000|1|7|alkali-metal|0.79|[Rn] 7s1|0.00|0.00|7|1
88|Ra|Radium|226.000000|2|7|alkaline-earth|0.90|[Rn] 7s2|0.00|0.00|7|2
89|Ac|Actinium|227.000000|0|7|actinide|1.10|[Rn] 6d1 7s2|1.88|0.00|9|3
90|Th|Thorium|232.037740|0|7|actinide|1.30|[Rn] 6d2 7s2|1.80|1.65|9|4
91|Pa|Protactinium|231.035882|0|7|actinide|1.50|[Rn] 5f2 6d1 7s2|1.61|0.00|9|5
92|U|Uranium|238.028913|0|7|actinide|1.38|[Rn] 5f3 6d1 7s2|1.38|1.42|9|6
93|Np|Neptunium|237.000000|0|7|actinide|1.36|[Rn] 5f4 6d1 7s2|1.30|0.00|9|7
94|Pu|Plutonium|244.000000|0|7|actinide|1.28|[Rn] 5f6 7s2|1.51|0.00|9|8
95|Am|Americium|243.000000|0|7|actinide|1.13|[Rn] 5f7 7s2|1.73|0.00|9|9
96|Cm|Curium|247.000000|0|7|actinide|1.28|[Rn] 5f7 6d1 7s2|2.99|0.00|9|10
97|Bk|Berkelium|247.000000|0|7|actinide|1.30|[Rn] 5f9 7s2|2.97|0.00|9|11
98|Cf|Californium|251.000000|0|7|actinide|1.30|[Rn] 5f10 7s2|2.95|0.00|9|12
99|Es|Einsteinium|252.000000|0|7|actinide|1.30|[Rn] 5f11 7s2|2.92|0.00|9|13
100|Fm|Fermium|257.000000|0|7|actinide|1.30|[Rn] 5f12 7s2|2.90|0.00|9|14
101|Md|Mendelevium|258.000000|0|7|actinide|1.30|[Rn] 5f13 7s2|2.87|0.00|9|15
102|No|Nobelium|259.000000|0|7|actinide|1.30|[Rn] 5f14 7s2|2.85|0.00|9|16
103|Lr|Lawrencium|266.000000|0|7|actinide|1.30|[Rn] 5f14 7s2 7p1|2.82|0.00|9|17
104|Rf|Rutherfordium|267.000000|4|7|transition-metal|0.00|[Rn] 5f14 6d2 7s2|0.00|0.00|7|4
105|Db|Dubnium|268.000000|5|7|transition-metal|0.00|[Rn] 5f14 6d3 7s2|0.00|0.00|7|5
106|Sg|Seaborgium|269.000000|6|7|transition-metal|0.00|[Rn] 5f14 6d4 7s2|0.00|0.00|7|6
107|Bh|Bohrium|270.000000|7|7|transition-metal|0.00|[Rn] 5f14 6d5 7s2|0.00|0.00|7|7
108|Hs|Hassium|269.000000|8|7|transition-metal|0.00|[Rn] 5f14 6d6 7s2|0.00|0.00|7|8
109|Mt|Meitnerium|278.000000|9|7|transition-metal|0.00|[Rn] 5f14 6d7 7s2|0.00|0.00|7|9
110|Ds|Darmstadtium|281.000000|10|7|transition-metal|0.00|[Rn] 5f14 6d9 7s1|0.00|0.00|7|10
111|Rg|Roentgenium|282.000000|11|7|transition-metal|0.00|[Rn] 5f14 6d10 7s1|0.00|0.00|7|11
112|Cn|Copernicium|285.000000|12|7|transition-metal|0.00|[Rn] 5f14 6d10 7s2|0.00|0.00|7|12
113|Nh|Nihonium|286.000000|13|7|transition-metal|0.00|[Rn] 5f14 6d10 7s2 7p1|0.00|0.00|7|13
114|Fl|Flerovium|289.000000|14|7|post-transition-metal|0.00|[Rn] 5f14 6d10 7s2 7p2|0.00|0.00|7|14
115|Mc|Moscovium|289.000000|15|7|post-transition-metal|0.00|[Rn] 5f14 6d10 7s2 7p3|0.00|0.00|7|15
116|Lv|Livermorium|293.000000|16|7|post-transition-metal|0.00|[Rn] 5f14 6d10 7s2 7p4|0.00|0.00|7|16
117|Ts|Tennessine|294.000000|17|7|halogen|0.00|[Rn] 5f14 6d10 7s2 7p5|0.00|0.00|7|17
118|Og|Oganesson|294.000000|18|7|noble-gas|0.00|[Rn] 5f14 6d10 7s2 7p6|0.00|0.00|7|18
`
