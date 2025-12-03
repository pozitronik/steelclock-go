package glyphs

// KatakanaGlyphs5x7 contains Japanese Katakana characters in 5x7 pixel format
// These are the characters commonly seen in the Matrix "digital rain" effect
var KatakanaGlyphs5x7 = map[rune]*Glyph{
	// ア (a)
	'ア': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, true, true, true, true},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	// イ (i)
	'イ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, true, false},
			{true, false, false, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
		},
	},
	// ウ (u)
	'ウ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
		},
	},
	// エ (e)
	'エ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{true, true, true, true, true},
		},
	},
	// オ (o)
	'オ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, true},
			{false, true, true, false, true},
			{true, false, true, false, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
		},
	},
	// カ (ka)
	'カ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, true},
			{false, true, false, false, true},
			{true, false, false, false, true},
			{false, false, false, true, false},
			{false, false, false, false, false},
		},
	},
	// キ (ki)
	'キ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
		},
	},
	// ク (ku)
	'ク': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, true, false},
			{true, true, true, true, true},
			{true, false, false, true, false},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
		},
	},
	// ケ (ke)
	'ケ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, false, false},
			{false, true, true, true, true},
			{true, true, false, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{false, false, true, false, false},
			{false, false, false, true, true},
		},
	},
	// コ (ko)
	'コ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
		},
	},
	// サ (sa)
	'サ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{false, true, false, true, false},
			{true, true, true, true, true},
			{false, true, false, true, false},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
		},
	},
	// シ (shi)
	'シ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, false},
			{true, false, false, false, true},
			{false, true, false, false, false},
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{true, true, false, false, false},
		},
	},
	// ス (su)
	'ス': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, true, false, true, false},
			{true, false, false, false, true},
			{false, false, false, false, false},
		},
	},
	// セ (se)
	'セ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, false, false},
			{false, true, true, true, true},
			{true, true, false, false, false},
			{false, true, false, false, false},
			{false, true, true, true, true},
			{false, true, false, false, false},
			{false, true, false, false, false},
		},
	},
	// ソ (so)
	'ソ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, true},
			{false, true, false, false, true},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
		},
	},
	// タ (ta)
	'タ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, true},
			{false, true, false, false, true},
			{true, false, false, true, false},
			{false, false, false, true, false},
			{false, false, true, false, false},
		},
	},
	// チ (chi)
	'チ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
		},
	},
	// ツ (tsu)
	'ツ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, true, false, true},
			{true, false, true, false, true},
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
		},
	},
	// テ (te)
	'テ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
		},
	},
	// ト (to)
	'ト': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, false, false},
			{true, false, false, true, false},
			{true, false, false, false, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
		},
	},
	// ナ (na)
	'ナ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	// ニ (ni)
	'ニ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, true, true, true, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{true, true, true, true, true},
			{false, false, false, false, false},
		},
	},
	// ヌ (nu)
	'ヌ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, false, true, false},
			{false, true, true, false, false},
			{false, false, true, false, false},
			{false, true, false, true, false},
			{true, false, false, false, true},
			{false, false, false, false, false},
		},
	},
	// ネ (ne)
	'ネ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, true, true, true, false},
			{true, false, true, false, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
		},
	},
	// ノ (no)
	'ノ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	// ハ (ha)
	'ハ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, true, false, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{false, false, false, false, false},
		},
	},
	// ヒ (hi)
	'ヒ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, false},
			{true, false, false, false, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{false, false, false, true, false},
		},
	},
	// フ (fu)
	'フ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
		},
	},
	// ヘ (he)
	'ヘ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, false, true, false, false},
			{false, true, false, true, false},
			{true, false, false, false, true},
			{false, false, false, false, false},
			{false, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	// ホ (ho)
	'ホ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, true, true, true, false},
			{true, false, true, false, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
		},
	},
	// マ (ma)
	'マ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
			{false, false, false, false, false},
		},
	},
	// ミ (mi)
	'ミ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{false, true, true, true, false},
			{false, false, false, false, false},
			{true, true, true, true, true},
			{false, false, false, false, false},
			{false, true, true, true, false},
			{false, false, false, false, false},
		},
	},
	// ム (mu)
	'ム': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{true, false, false, true, false},
			{true, true, true, true, true},
			{false, false, false, false, false},
		},
	},
	// メ (me)
	'メ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, true, true, false, false},
			{false, false, true, false, false},
			{false, true, false, true, false},
			{true, false, false, false, true},
			{false, false, false, false, false},
		},
	},
	// モ (mo)
	'モ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, true, false, false},
			{true, true, true, true, true},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, true, false, false},
			{false, false, false, true, true},
		},
	},
	// ヤ (ya)
	'ヤ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, false, false},
			{false, true, true, true, true},
			{true, true, false, true, false},
			{false, true, false, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
		},
	},
	// ユ (yu)
	'ユ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, false, false, false, false},
			{true, true, true, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{true, true, true, true, true},
			{false, false, false, false, false},
		},
	},
	// ヨ (yo)
	'ヨ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
			{false, false, false, false, false},
		},
	},
	// ラ (ra)
	'ラ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
		},
	},
	// リ (ri)
	'リ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, true, false},
			{true, false, false, true, false},
			{true, false, false, true, false},
			{true, false, false, true, false},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
		},
	},
	// ル (ru)
	'ル': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{false, true, false, true, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{false, true, false, true, false},
			{true, false, false, false, true},
			{false, false, false, false, false},
		},
	},
	// レ (re)
	'レ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, false, false, false, true},
			{true, false, false, true, false},
			{false, true, true, false, false},
		},
	},
	// ロ (ro)
	'ロ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
			{false, false, false, false, false},
		},
	},
	// ワ (wa)
	'ワ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{true, false, false, false, true},
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
		},
	},
	// ヲ (wo)
	'ヲ': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, true, true, true, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
		},
	},
	// ン (n)
	'ン': {
		Width: 5, Height: 7,
		Data: [][]bool{
			{true, false, false, false, false},
			{false, true, false, false, true},
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{true, false, false, false, false},
		},
	},
}

// KatakanaGlyphs3x5 contains Japanese Katakana characters in 3x5 pixel format
// Simplified versions for small displays
var KatakanaGlyphs3x5 = map[rune]*Glyph{
	// ア (a)
	'ア': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, false, true},
			{false, true, true},
			{true, false, false},
			{false, false, false},
		},
	},
	// イ (i)
	'イ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, true},
			{false, true, true},
			{true, false, true},
			{false, false, true},
			{false, false, true},
		},
	},
	// ウ (u)
	'ウ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{true, false, true},
			{false, false, true},
			{false, true, false},
		},
	},
	// エ (e)
	'エ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, true, false},
			{false, true, false},
			{false, true, false},
			{true, true, true},
		},
	},
	// オ (o)
	'オ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{false, true, true},
			{true, true, false},
			{false, true, false},
		},
	},
	// カ (ka)
	'カ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{false, true, true},
			{true, false, true},
			{false, false, false},
		},
	},
	// キ (ki)
	'キ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{false, true, false},
			{true, true, true},
			{false, true, false},
		},
	},
	// ク (ku)
	'ク': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, true},
			{true, true, true},
			{true, false, true},
			{false, true, false},
			{true, false, false},
		},
	},
	// ケ (ke)
	'ケ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{true, true, true},
			{true, true, false},
			{true, false, false},
			{false, true, true},
		},
	},
	// コ (ko)
	'コ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, false, true},
			{false, false, true},
			{false, false, true},
			{true, true, true},
		},
	},
	// サ (sa)
	'サ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, true, true},
			{true, false, true},
			{false, false, true},
			{false, true, false},
		},
	},
	// シ (shi)
	'シ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{true, false, true},
			{false, false, true},
			{false, true, false},
			{true, false, false},
		},
	},
	// ス (su)
	'ス': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, false, true},
			{false, true, false},
			{true, false, true},
			{false, false, false},
		},
	},
	// セ (se)
	'セ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{true, true, true},
			{true, true, false},
			{true, true, true},
			{true, false, false},
		},
	},
	// ソ (so)
	'ソ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{false, false, true},
			{false, true, false},
			{true, false, false},
			{false, false, false},
		},
	},
	// タ (ta)
	'タ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{false, true, true},
			{true, false, true},
			{false, true, false},
		},
	},
	// チ (chi)
	'チ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, true, false},
			{true, true, true},
			{false, true, false},
			{true, false, false},
		},
	},
	// ツ (tsu)
	'ツ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{false, false, true},
			{false, true, false},
			{true, false, false},
		},
	},
	// テ (te)
	'テ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, true, false},
			{true, true, true},
			{false, true, false},
			{true, false, false},
		},
	},
	// ト (to)
	'ト': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{true, true, false},
			{true, false, true},
			{true, false, false},
			{true, false, false},
		},
	},
	// ナ (na)
	'ナ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{false, true, false},
			{true, false, false},
			{false, false, false},
		},
	},
	// ニ (ni)
	'ニ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, false},
			{false, false, false},
			{true, true, true},
			{false, false, false},
		},
	},
	// ヌ (nu)
	'ヌ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, true, false},
			{false, true, false},
			{true, false, true},
			{false, false, false},
		},
	},
	// ネ (ne)
	'ネ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{true, true, true},
			{false, true, false},
			{false, true, false},
		},
	},
	// ノ (no)
	'ノ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, true},
			{false, true, false},
			{false, true, false},
			{true, false, false},
			{false, false, false},
		},
	},
	// ハ (ha)
	'ハ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, false},
			{true, false, true},
			{true, false, true},
			{false, false, false},
		},
	},
	// ヒ (hi)
	'ヒ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{true, true, true},
			{true, false, true},
			{false, false, true},
			{false, true, false},
		},
	},
	// フ (fu)
	'フ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, false, true},
			{false, true, false},
			{true, false, false},
			{false, false, false},
		},
	},
	// ヘ (he)
	'ヘ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{false, true, false},
			{true, false, true},
			{false, false, false},
			{false, false, false},
		},
	},
	// ホ (ho)
	'ホ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{true, true, true},
			{true, true, true},
			{false, true, false},
			{false, true, false},
		},
	},
	// マ (ma)
	'マ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, false, true},
			{false, true, false},
			{true, false, false},
			{false, false, false},
		},
	},
	// ミ (mi)
	'ミ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, true},
			{false, false, false},
			{true, true, true},
			{false, false, false},
			{false, true, true},
		},
	},
	// ム (mu)
	'ム': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, true, false},
			{false, true, false},
			{true, false, false},
			{true, false, true},
			{true, true, true},
		},
	},
	// メ (me)
	'メ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, true},
			{false, true, false},
			{false, true, false},
			{true, false, true},
			{false, false, false},
		},
	},
	// モ (mo)
	'モ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, true, false},
			{true, true, true},
			{false, true, false},
			{false, false, true},
		},
	},
	// ヤ (ya)
	'ヤ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{true, true, true},
			{true, true, false},
			{false, true, false},
			{false, true, false},
		},
	},
	// ユ (yu)
	'ユ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{false, false, false},
			{true, true, false},
			{false, true, false},
			{false, true, false},
			{true, true, true},
		},
	},
	// ヨ (yo)
	'ヨ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, false, true},
			{true, true, true},
			{false, false, true},
			{true, true, true},
		},
	},
	// ラ (ra)
	'ラ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, false, true},
			{true, true, true},
			{false, true, false},
			{true, false, false},
		},
	},
	// リ (ri)
	'リ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{true, false, true},
			{false, false, true},
			{false, true, false},
		},
	},
	// ル (ru)
	'ル': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, true},
			{true, false, true},
			{true, false, true},
			{true, false, true},
			{true, true, false},
		},
	},
	// レ (re)
	'レ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{true, false, false},
			{true, false, false},
			{true, false, true},
			{false, true, false},
		},
	},
	// ロ (ro)
	'ロ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{true, false, true},
			{true, false, true},
			{true, false, true},
			{true, true, true},
		},
	},
	// ワ (wa)
	'ワ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{true, false, true},
			{false, false, true},
			{false, true, false},
			{true, false, false},
		},
	},
	// ヲ (wo)
	'ヲ': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, true, true},
			{false, false, true},
			{true, true, true},
			{false, true, false},
			{true, false, false},
		},
	},
	// ン (n)
	'ン': {
		Width: 3, Height: 5,
		Data: [][]bool{
			{true, false, false},
			{false, false, true},
			{false, true, false},
			{true, false, false},
			{false, false, false},
		},
	},
}

// init adds katakana glyphs to Font5x7 and Font3x5
func init() {
	for r, g := range KatakanaGlyphs5x7 {
		Font5x7.Glyphs[r] = g
	}
	for r, g := range KatakanaGlyphs3x5 {
		Font3x5.Glyphs[r] = g
	}
}
