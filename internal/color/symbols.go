package color

// Symbols holds the glyphs tfsum uses to mark change kinds, falling back
// to ASCII equivalents when the terminal's locale doesn't advertise UTF-8.
type Symbols struct {
	Create    string
	Update    string
	Replace   string
	Destroy   string
	Drift     string
	Unknown   string
	Sensitive string
	NoOp      string
	Cross     string

	Bullet       string
	Arrow        string
	ModuleMarker string

	BoxTopLeft     string
	BoxTopRight    string
	BoxBottomLeft  string
	BoxBottomRight string
	BoxHorizontal  string
	BoxVertical    string
	Rule           string
}

func symbolsFor(unicode bool) Symbols {
	if unicode {
		return Symbols{
			Create:    "+",
			Update:    "~",
			Replace:   "-/+",
			Destroy:   "-",
			Drift:     "!",
			Unknown:   "?",
			Sensitive: "***",
			NoOp:      "\u00b7",
			Cross:     "\u2716",

			Bullet:       "\u2022",
			Arrow:        "\u2192",
			ModuleMarker: "\u25b8",

			BoxTopLeft:     "\u250c",
			BoxTopRight:    "\u2510",
			BoxBottomLeft:  "\u2514",
			BoxBottomRight: "\u2518",
			BoxHorizontal:  "\u2500",
			BoxVertical:    "\u2502",
			Rule:           "\u2500",
		}
	}
	return Symbols{
		Create:    "+",
		Update:    "~",
		Replace:   "-/+",
		Destroy:   "-",
		Drift:     "!",
		Unknown:   "?",
		Sensitive: "***",
		NoOp:      ".",
		Cross:     "x",

		Bullet:       "*",
		Arrow:        "->",
		ModuleMarker: ">",

		BoxTopLeft:     "+",
		BoxTopRight:    "+",
		BoxBottomLeft:  "+",
		BoxBottomRight: "+",
		BoxHorizontal:  "-",
		BoxVertical:    "|",
		Rule:           "-",
	}
}
