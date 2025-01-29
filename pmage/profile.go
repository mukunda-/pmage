package pmage

const SystemSnes = "snes"

// A profile is the global configuration for the conversion process, specified at the
// command line.
type Profile struct {
	System string
}

func (p *Profile) IsValidBpp(bpp int16) bool {
	if p.System == SystemSnes {
		// SNES BG modes support 4-color, 16-color and 256-color.
		//
		// 16bit may be usedful for generating non-indexed images that are not used
		// directly.
		return bpp == 2 || bpp == 4 || bpp == 8 || bpp == 16
	}
	panic("unknown system")
}

func (p *Profile) GetColorFormat() ColorFormat {
	if p.System == SystemSnes {
		return ColorFormat15bgr
	}
	panic("unknown system")
}

func (p *Profile) DefaultBpp() int16 {
	if p.System == SystemSnes {
		return 4
	}
	panic("unknown system")
}

func (p *Profile) DefaultSegment() string {
	return "GRAPHICS"
}
