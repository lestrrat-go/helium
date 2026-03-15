package xslt3

import (
	"strconv"
	"strings"
	"unicode"
)

// numberToWordsLang converts a number to words in the given language and case.
// caseMode is "lower", "upper", or "title".
// ordinal is the ordinal hint (e.g., "yes", "%spellout-ordinal-masculine").
func numberToWordsLang(n int, caseMode string, lang string, ordinal string) string {
	// Normalize language tag to base language
	baseLang := strings.ToLower(lang)
	if idx := strings.IndexAny(baseLang, "-_"); idx >= 0 {
		baseLang = baseLang[:idx]
	}

	var result string
	if ordinal != "" {
		result = numberToOrdinalWords(n, baseLang, ordinal)
	} else {
		result = numberToCardinalWords(n, baseLang)
	}

	switch caseMode {
	case "upper":
		return strings.ToUpper(result)
	case "title":
		return toTitleCase(result)
	default:
		return result
	}
}

func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// numberToCardinalWords returns a cardinal number word in the given language.
func numberToCardinalWords(n int, lang string) string {
	switch lang {
	case "de":
		return germanCardinal(n)
	case "it":
		return italianCardinal(n)
	case "fr":
		return frenchCardinal(n)
	default:
		return numberToWords(n, false)
	}
}

// numberToOrdinalWords returns an ordinal number word in the given language.
func numberToOrdinalWords(n int, lang string, ordinal string) string {
	switch lang {
	case "de":
		return germanOrdinal(n, ordinal)
	case "it":
		return italianOrdinal(n, ordinal)
	case "fr":
		return frenchOrdinal(n, ordinal)
	default:
		return englishOrdinal(n)
	}
}

// ============================================================
// English ordinals
// ============================================================

func englishOrdinal(n int) string {
	if n == 0 {
		return "zeroth"
	}
	if n < 0 {
		return "minus " + englishOrdinal(-n)
	}
	// For compound numbers, only the last component is ordinal
	if n >= 1000000000000 {
		q, r := n/1000000000000, n%1000000000000
		if r == 0 {
			return numberToWords(q, false) + " trillionth"
		}
		return numberToWords(q, false) + " trillion " + englishOrdinal(r)
	}
	if n >= 1000000000 {
		q, r := n/1000000000, n%1000000000
		if r == 0 {
			return numberToWords(q, false) + " billionth"
		}
		return numberToWords(q, false) + " billion " + englishOrdinal(r)
	}
	if n >= 1000000 {
		q, r := n/1000000, n%1000000
		if r == 0 {
			return numberToWords(q, false) + " millionth"
		}
		return numberToWords(q, false) + " million " + englishOrdinal(r)
	}
	if n >= 1000 {
		q, r := n/1000, n%1000
		if r == 0 {
			return numberToWords(q, false) + " thousandth"
		}
		return numberToWords(q, false) + " thousand " + englishOrdinal(r)
	}
	if n >= 100 {
		q, r := n/100, n%100
		if r == 0 {
			return numberToWords(q, false) + " hundredth"
		}
		return numberToWords(q, false) + " hundred and " + englishOrdinal(r)
	}
	if n >= 20 {
		q, r := n/10, n%10
		if r == 0 {
			return englishTensOrdinal(q)
		}
		tens := []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}
		return tens[q] + "-" + englishOnesOrdinal(r)
	}
	return englishOnesOrdinal(n)
}

func englishOnesOrdinal(n int) string {
	ords := []string{
		"zeroth", "first", "second", "third", "fourth", "fifth",
		"sixth", "seventh", "eighth", "ninth", "tenth",
		"eleventh", "twelfth", "thirteenth", "fourteenth", "fifteenth",
		"sixteenth", "seventeenth", "eighteenth", "nineteenth",
	}
	if n >= 0 && n < len(ords) {
		return ords[n]
	}
	return strconv.Itoa(n) + "th"
}

func englishTensOrdinal(n int) string {
	ords := []string{"", "", "twentieth", "thirtieth", "fortieth", "fiftieth",
		"sixtieth", "seventieth", "eightieth", "ninetieth"}
	if n >= 0 && n < len(ords) {
		return ords[n]
	}
	return strconv.Itoa(n*10) + "th"
}

// ============================================================
// German
// ============================================================

func germanCardinal(n int) string {
	if n == 0 {
		return "null"
	}
	if n < 0 {
		return "minus " + germanCardinal(-n)
	}
	ones := []string{"", "eins", "zwei", "drei", "vier", "fünf", "sechs", "sieben", "acht", "neun",
		"zehn", "elf", "zwölf", "dreizehn", "vierzehn", "fünfzehn", "sechzehn", "siebzehn", "achtzehn", "neunzehn"}
	tens := []string{"", "", "zwanzig", "dreißig", "vierzig", "fünfzig", "sechzig", "siebzig", "achtzig", "neunzig"}

	if n < 20 {
		return ones[n]
	}
	if n < 100 {
		if n%10 == 0 {
			return tens[n/10]
		}
		o := ones[n%10]
		if o == "eins" {
			o = "ein"
		}
		return o + "und" + tens[n/10]
	}
	if n < 1000 {
		w := ones[n/100]
		if w == "eins" {
			w = "ein"
		}
		w += "hundert"
		if n%100 != 0 {
			w += germanCardinal(n % 100)
		}
		return w
	}
	if n < 1000000 {
		q := n / 1000
		w := ""
		if q == 1 {
			w = "ein"
		} else {
			w = germanCardinal(q)
		}
		w += "tausend"
		if n%1000 != 0 {
			w += germanCardinal(n % 1000)
		}
		return w
	}
	if n < 1000000000 {
		q := n / 1000000
		w := ""
		if q == 1 {
			w = "eine Million "
		} else {
			w = germanCardinal(q) + " Millionen "
		}
		if n%1000000 != 0 {
			w += germanCardinal(n % 1000000)
		}
		return w
	}
	return strconv.Itoa(n)
}

func germanOrdinal(n int, _ string) string {
	// German ordinals append -te (1-19) or -ste (20+) to the cardinal stem
	if n <= 0 {
		return germanCardinal(n)
	}
	card := germanCardinal(n)
	if n < 20 {
		return card + "te"
	}
	return card + "ste"
}

// ============================================================
// Italian
// ============================================================

func italianCardinal(n int) string {
	if n == 0 {
		return "zero"
	}
	if n < 0 {
		return "meno " + italianCardinal(-n)
	}

	ones := []string{"", "uno", "due", "tre", "quattro", "cinque", "sei", "sette", "otto", "nove",
		"dieci", "undici", "dodici", "tredici", "quattordici", "quindici", "sedici", "diciassette", "diciotto", "diciannove"}
	tens := []string{"", "", "venti", "trenta", "quaranta", "cinquanta", "sessanta", "settanta", "ottanta", "novanta"}

	if n < 20 {
		return ones[n]
	}
	if n < 100 {
		t := tens[n/10]
		o := n % 10
		if o == 0 {
			return t
		}
		// Drop final vowel of tens before uno/otto
		if o == 1 || o == 8 {
			t = t[:len(t)-1]
		}
		return t + ones[o]
	}
	if n < 1000 {
		q := n / 100
		w := ""
		if q == 1 {
			w = "cento"
		} else {
			w = ones[q] + "cento"
		}
		r := n % 100
		if r != 0 {
			w += italianCardinal(r)
		}
		return w
	}
	if n < 1000000 {
		q := n / 1000
		w := ""
		if q == 1 {
			w = "mille"
		} else {
			w = italianCardinal(q) + "mila"
		}
		r := n % 1000
		if r != 0 {
			w += italianCardinal(r)
		}
		return w
	}
	return strconv.Itoa(n)
}

func italianOrdinal(n int, ordinal string) string {
	if n <= 0 {
		return italianCardinal(n)
	}

	// Detect gender from ordinal hint
	feminine := strings.Contains(ordinal, "feminine")

	// Italian ordinals 1-10 are irregular
	if n <= 10 {
		if feminine {
			ords := []string{"", "prima", "seconda", "terza", "quarta", "quinta",
				"sesta", "settima", "ottava", "nona", "decima"}
			return ords[n]
		}
		ords := []string{"", "primo", "secondo", "terzo", "quarto", "quinto",
			"sesto", "settimo", "ottavo", "nono", "decimo"}
		return ords[n]
	}

	// Regular Italian ordinals: cardinal stem + -esimo/-esima
	card := italianCardinal(n)
	// Drop final vowel before -esimo
	runes := []rune(card)
	last := runes[len(runes)-1]
	if last == 'a' || last == 'e' || last == 'i' || last == 'o' || last == 'u' {
		card = string(runes[:len(runes)-1])
	}
	if feminine {
		return card + "esima"
	}
	return card + "esimo"
}

// ============================================================
// French
// ============================================================

func frenchCardinal(n int) string {
	if n == 0 {
		return "zéro"
	}
	if n < 0 {
		return "moins " + frenchCardinal(-n)
	}

	ones := []string{"", "un", "deux", "trois", "quatre", "cinq", "six", "sept", "huit", "neuf",
		"dix", "onze", "douze", "treize", "quatorze", "quinze", "seize", "dix-sept", "dix-huit", "dix-neuf"}

	if n < 20 {
		return ones[n]
	}
	if n < 100 {
		tens := []string{"", "", "vingt", "trente", "quarante", "cinquante", "soixante", "soixante", "quatre-vingt", "quatre-vingt"}
		t := n / 10
		o := n % 10
		if t == 7 || t == 9 {
			// 70-79 = soixante-dix...; 90-99 = quatre-vingt-dix...
			o += 10
		}
		if o == 0 {
			if t == 8 {
				return "quatre-vingts"
			}
			return tens[t]
		}
		sep := "-"
		if o == 1 && t != 8 && t != 9 {
			sep = " et "
		}
		if o == 11 && t == 7 {
			sep = " et "
		}
		return tens[t] + sep + ones[o]
	}
	if n < 1000 {
		q := n / 100
		w := ""
		if q == 1 {
			w = "cent"
		} else {
			w = ones[q] + " cent"
		}
		r := n % 100
		if r == 0 && q > 1 {
			w += "s"
		}
		if r != 0 {
			w += " " + frenchCardinal(r)
		}
		return w
	}
	if n < 1000000 {
		q := n / 1000
		w := ""
		if q == 1 {
			w = "mille"
		} else {
			w = frenchCardinal(q) + " mille"
		}
		r := n % 1000
		if r != 0 {
			w += " " + frenchCardinal(r)
		}
		return w
	}
	return strconv.Itoa(n)
}

func frenchOrdinal(n int, _ string) string {
	if n == 1 {
		return "premier"
	}
	if n <= 0 {
		return frenchCardinal(n)
	}
	card := frenchCardinal(n)
	// Add -ième suffix (drop trailing -e if present)
	if strings.HasSuffix(card, "e") {
		card = card[:len(card)-1]
	}
	return card + "ième"
}
