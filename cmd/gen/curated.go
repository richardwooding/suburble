package main

// mergeDef assembles one colloquial suburb from multiple official planning
// suburbs — the layer shatters famous areas into slivers (Bloubergstrand is
// officially a 0.84 km² beachfront strip; the Bloubergstrand everyone knows
// includes Blouberg Rise/Sands/Rant). Merged suburbs render as one
// multi-part silhouette and join the guess list; their members remain
// individually guessable and stay in the hard-mode answer pool.
type mergeDef struct {
	Name    string
	Members []string // explicit official names
	Prefix  string   // plus every official name with this prefix
}

var merges = []mergeDef{
	{Name: "BLOUBERGSTRAND", Members: []string{
		"BLAAUWBERGSTRAND", "BLOUBERG SANDS", "BLOUBERG RISE", "BLOUBERGRANT",
		"WEST BEACH", // the connective tissue between the beachfront strip and the inland blocks
	}},
	{Name: "BELHAR", Prefix: "BELHAR "}, // EXT 1..23
	{Name: "DELFT", Prefix: "DELFT "},   // DELFT 1..9, DELFT SOUTH
	{Name: "MITCHELLS PLAIN", Members: []string{
		"MITCHELLS PLAIN CBD", "WESTRIDGE - MITCHELLS PLAIN", "ROCKLANDS",
		"TAFELSIG", "LENTEGEUR", "BEACON VALLEY", "PORTLAND", "EASTRIDGE",
		"WOODLANDS", "NEW WOODLANDS", "WESTGATE", "COLORADO PARK",
	}},
}

// curated lists the well-known suburbs eligible as normal-mode answers.
// Names must match the dataset's OFC_SBRB_NAME values exactly (all caps) or
// a merged colloquial name above; gen fails if any entry is missing, so
// drift is caught at generation time. Everything else stays guessable — and
// available in hard mode.
var curated = []string{
	// City Bowl & Atlantic Seaboard
	"BANTRY BAY", "BO-KAAP", "CAMPS BAY / BAKOVEN", "CLIFTON", "DISTRICT SIX",
	"FORESHORE", "FRESNAYE", "GARDENS", "GREEN POINT",
	"HOUT BAY", "LLANDUDNO", "MOUILLE POINT", "ORANJEZICHT",
	"SEA POINT", "TAMBOERSKLOOF", "THREE ANCHOR BAY", "VREDEHOEK", "WOODSTOCK",
	"SALT RIVER",
	// Southern Suburbs
	"BERGVLIET", "BISHOPSCOURT", "CLAREMONT", "CONSTANTIA", "DIEP RIVER",
	"HEATHFIELD", "KENILWORTH", "KIRSTENHOF", "LANSDOWNE",
	"MEADOWRIDGE", "MOWBRAY", "NEWLANDS", "OBSERVATORY", "PLUMSTEAD",
	"RONDEBOSCH", "RONDEBOSCH EAST", "ROSEBANK", "SOUTHFIELD", "TOKAI",
	"WETTON", "WYNBERG",
	// Deep South / False Bay
	"CLOVELLY", "FISH HOEK", "GLENCAIRN", "KALK BAY", "KOMMETJIE",
	"LAKESIDE", "MUIZENBERG", "NOORDHOEK", "OCEAN VIEW",
	"SCARBOROUGH", "SIMON'S TOWN", "ST JAMES", "SUN VALLEY", "SUNNYDALE",
	// Cape Flats & metro south-east
	"ATHLONE", "BONTEHEUWEL", "BRIDGETOWN", "CROSSROADS",
	"BELHAR", "DELFT",
	"GRASSY PARK", "GUGULETU", "HANOVER PARK", "KHAYELITSHA", "LANGA",
	"LAVENDER HILL", "LOTUS RIVER", "MANENBERG", "MITCHELLS PLAIN", "NYANGA",
	"OTTERY", "PHILIPPI", "RETREAT", "STEENBERG", "STRANDFONTEIN",
	// Northern Suburbs
	"BELLVILLE CBD", "BELLVILLE SOUTH", "BOTHASIG", "BRACKENFELL CENTRAL",
	"BURGUNDY ESTATE", "DURBANVILLE", "EDGEMEAD", "ELSIES RIVER",
	"GOODWOOD ESTATE", "KRAAIFONTEIN", "MONTE VISTA", "PANORAMA", "PAROW",
	"PINELANDS", "PLATTEKLOOF 1", "RAVENSMEAD", "RICHWOOD", "THORNTON",
	"WELGEMOED",
	// Blaauwberg / West Coast
	"BIG BAY", "BLOUBERGSTRAND", "BROOKLYN", "CENTURY CITY",
	"KENSINGTON", "MAITLAND", "MELKBOSCH STRAND", "MILNERTON", "PAARDEN EILAND",
	"PARKLANDS", "RUGBY", "SUNNINGDALE", "SUNSET BEACH", "TABLE VIEW",
	"YSTERPLAAT",
	// Helderberg & far east
	"EERSTERIVIER", "GORDONS BAY", "MACASSAR",
	"SOMERSET WEST", "STRAND",
}
