package main

// UI fonts, embedded. Both families are licensed under the SIL Open Font
// License 1.1 (see assets/fonts/OFL-*.txt), which permits bundling them in
// commercial games.
//   Lilita One — display/headings (Juan Montoreano)
//   Fira Sans  — interface text (Mozilla / Carrois)

import _ "embed"

var (
	//go:embed assets/fonts/FiraSans-SemiBold.ttf
	fontRegular []byte
	//go:embed assets/fonts/FiraSans-Bold.ttf
	fontBold []byte
	//go:embed assets/fonts/LilitaOne-Regular.ttf
	fontDisplay []byte
)
