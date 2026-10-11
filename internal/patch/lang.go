package patch

import "strings"

// Text holds the words shown in the game, in one language. Medals go bronze, silver, gold, diamond.
type Text struct {
	Title        string
	NoTrophy     string
	Sure         [4]string // the medal sought is guaranteed
	Possible     [4]string
	Impossible   [4]string
	TiersShort   [4]string
	Percent      string // "%" or " %"
	SkillMissing string // followed by the skill name (Spotting Knowledge, level 3)
}

// SkillName is the skill needed for the gauge, when the game's own name cannot be read.
const SkillName = "Spotting Knowledge 3"

var texts = map[string]Text{
	"en": {"POTENTIAL", "NO TROPHY",
		[4]string{"BRONZE GUARANTEED", "SILVER GUARANTEED", "GOLD GUARANTEED", "DIAMOND GUARANTEED"},
		[4]string{"BRONZE POSSIBLE", "SILVER POSSIBLE", "GOLD POSSIBLE", "DIAMOND POSSIBLE"},
		[4]string{"BRONZE IMPOSSIBLE", "SILVER IMPOSSIBLE", "GOLD IMPOSSIBLE", "DIAMOND IMPOSSIBLE"},
		[4]string{"BR.", "SIL.", "GOLD", "DIA."}, "%", "Skill missing: "},
	"fr": {"POTENTIEL", "PAS DE TROPHÉE",
		[4]string{"BRONZE GARANTI", "ARGENT GARANTI", "OR GARANTI", "DIAMANT GARANTI"},
		[4]string{"BRONZE POSSIBLE", "ARGENT POSSIBLE", "OR POSSIBLE", "DIAMANT POSSIBLE"},
		[4]string{"BRONZE IMPOSSIBLE", "ARGENT IMPOSSIBLE", "OR IMPOSSIBLE", "DIAMANT IMPOSSIBLE"},
		[4]string{"BR.", "ARG.", "OR", "DIA."}, " %", "Compétence absente : "},
	"de": {"POTENZIAL", "KEINE TROPHÄE",
		[4]string{"BRONZE SICHER", "SILBER SICHER", "GOLD SICHER", "DIAMANT SICHER"},
		[4]string{"BRONZE MÖGLICH", "SILBER MÖGLICH", "GOLD MÖGLICH", "DIAMANT MÖGLICH"},
		[4]string{"BRONZE UNMÖGLICH", "SILBER UNMÖGLICH", "GOLD UNMÖGLICH", "DIAMANT UNMÖGLICH"},
		[4]string{"BR.", "SILB.", "GOLD", "DIA."}, " %", "Fertigkeit fehlt: "},
	"es": {"POTENCIAL", "SIN TROFEO",
		[4]string{"BRONCE ASEGURADO", "PLATA ASEGURADA", "ORO ASEGURADO", "DIAMANTE ASEGURADO"},
		[4]string{"BRONCE POSIBLE", "PLATA POSIBLE", "ORO POSIBLE", "DIAMANTE POSIBLE"},
		[4]string{"BRONCE IMPOSIBLE", "PLATA IMPOSIBLE", "ORO IMPOSIBLE", "DIAMANTE IMPOSIBLE"},
		[4]string{"BR.", "PLATA", "ORO", "DIAM."}, " %", "Habilidad ausente: "},
	"it": {"POTENZIALE", "NESSUN TROFEO",
		[4]string{"BRONZO GARANTITO", "ARGENTO GARANTITO", "ORO GARANTITO", "DIAMANTE GARANTITO"},
		[4]string{"BRONZO POSSIBILE", "ARGENTO POSSIBILE", "ORO POSSIBILE", "DIAMANTE POSSIBILE"},
		[4]string{"BRONZO IMPOSSIBILE", "ARGENTO IMPOSSIBILE", "ORO IMPOSSIBILE", "DIAMANTE IMPOSSIBILE"},
		[4]string{"BR.", "ARG.", "ORO", "DIAM."}, "%", "Abilità assente: "},
	"pl": {"POTENCJAŁ", "BRAK TROFEUM",
		[4]string{"BRĄZ PEWNY", "SREBRO PEWNE", "ZŁOTO PEWNE", "DIAMENT PEWNY"},
		[4]string{"BRĄZ MOŻLIWY", "SREBRO MOŻLIWE", "ZŁOTO MOŻLIWE", "DIAMENT MOŻLIWY"},
		[4]string{"BRĄZ NIEMOŻLIWY", "SREBRO NIEMOŻLIWE", "ZŁOTO NIEMOŻLIWE", "DIAMENT NIEMOŻLIWY"},
		[4]string{"BR.", "SREB.", "ZŁOTO", "DIAM."}, "%", "Brak umiejętności: "},
	"ru": {"ПОТЕНЦИАЛ", "НЕТ ТРОФЕЯ",
		[4]string{"БРОНЗА ГАРАНТИРОВАНА", "СЕРЕБРО ГАРАНТИРОВАНО", "ЗОЛОТО ГАРАНТИРОВАНО", "АЛМАЗ ГАРАНТИРОВАН"},
		[4]string{"БРОНЗА ВОЗМОЖНА", "СЕРЕБРО ВОЗМОЖНО", "ЗОЛОТО ВОЗМОЖНО", "АЛМАЗ ВОЗМОЖЕН"},
		[4]string{"БРОНЗА НЕВОЗМОЖНА", "СЕРЕБРО НЕВОЗМОЖНО", "ЗОЛОТО НЕВОЗМОЖНО", "АЛМАЗ НЕВОЗМОЖЕН"},
		[4]string{"БР.", "СЕР.", "ЗОЛ.", "АЛМ."}, "%", "Нет навыка: "},
	"pt": {"POTENCIAL", "SEM TROFÉU",
		[4]string{"BRONZE GARANTIDO", "PRATA GARANTIDA", "OURO GARANTIDO", "DIAMANTE GARANTIDO"},
		[4]string{"BRONZE POSSÍVEL", "PRATA POSSÍVEL", "OURO POSSÍVEL", "DIAMANTE POSSÍVEL"},
		[4]string{"BRONZE IMPOSSÍVEL", "PRATA IMPOSSÍVEL", "OURO IMPOSSÍVEL", "DIAMANTE IMPOSSÍVEL"},
		[4]string{"BR.", "PRATA", "OURO", "DIAM."}, "%", "Habilidade ausente: "},
	"cs": {"POTENCIÁL", "BEZ TROFEJE",
		[4]string{"BRONZ JISTÝ", "STŘÍBRO JISTÉ", "ZLATO JISTÉ", "DIAMANT JISTÝ"},
		[4]string{"BRONZ MOŽNÝ", "STŘÍBRO MOŽNÉ", "ZLATO MOŽNÉ", "DIAMANT MOŽNÝ"},
		[4]string{"BRONZ NEMOŽNÝ", "STŘÍBRO NEMOŽNÉ", "ZLATO NEMOŽNÉ", "DIAMANT NEMOŽNÝ"},
		[4]string{"BR.", "STŘ.", "ZLATO", "DIA."}, " %", "Chybí dovednost: "},
	"ja": {"ポテンシャル", "トロフィーなし",
		[4]string{"ブロンズ確定", "シルバー確定", "ゴールド確定", "ダイヤ確定"},
		[4]string{"ブロンズ可能", "シルバー可能", "ゴールド可能", "ダイヤ可能"},
		[4]string{"ブロンズ不可", "シルバー不可", "ゴールド不可", "ダイヤ不可"},
		[4]string{"ブロンズ", "シルバー", "ゴールド", "ダイヤ"}, "%", "スキル未習得："},
	"zh-hans": {"潜力", "无奖杯",
		[4]string{"必定铜牌", "必定银牌", "必定金牌", "必定钻石"},
		[4]string{"可能铜牌", "可能银牌", "可能金牌", "可能钻石"},
		[4]string{"不可能铜牌", "不可能银牌", "不可能金牌", "不可能钻石"},
		[4]string{"铜", "银", "金", "钻"}, "%", "缺少技能："},
	"zh-hant": {"潛力", "無獎盃",
		[4]string{"必定銅牌", "必定銀牌", "必定金牌", "必定鑽石"},
		[4]string{"可能銅牌", "可能銀牌", "可能金牌", "可能鑽石"},
		[4]string{"不可能銅牌", "不可能銀牌", "不可能金牌", "不可能鑽石"},
		[4]string{"銅", "銀", "金", "鑽"}, "%", "缺少技能："},
	"ko": {"잠재력", "트로피 없음",
		[4]string{"브론즈 확정", "실버 확정", "골드 확정", "다이아몬드 확정"},
		[4]string{"브론즈 가능", "실버 가능", "골드 가능", "다이아몬드 가능"},
		[4]string{"브론즈 불가", "실버 불가", "골드 불가", "다이아몬드 불가"},
		[4]string{"브론즈", "실버", "골드", "다이아"}, "%", "스킬 없음: "},
}

var steamLanguages = map[string]string{
	"english": "en", "french": "fr", "german": "de", "spanish": "es", "latam": "es", "italian": "it",
	"polish": "pl", "russian": "ru", "brazilian": "pt", "portuguese": "pt", "czech": "cs", "japanese": "ja",
	"schinese": "zh-hans", "tchinese": "zh-hant", "koreana": "ko",
}

// Languages lists the available language codes.
func Languages() []string {
	return []string{"en", "fr", "de", "es", "it", "pl", "ru", "pt", "cs", "ja", "zh-hans", "zh-hant", "ko"}
}

// LanguageFor maps a Steam language name or a language code to a supported code (default "en").
func LanguageFor(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if _, ok := texts[n]; ok {
		return n
	}
	if c, ok := steamLanguages[n]; ok {
		return c
	}
	// language codes with a region, as Windows and the Epic Games Launcher give them ("fr-FR")
	switch {
	case strings.HasPrefix(n, "zh"):
		if strings.Contains(n, "hant") || strings.HasSuffix(n, "-tw") || strings.HasSuffix(n, "-hk") || strings.HasSuffix(n, "-mo") {
			return "zh-hant"
		}
		return "zh-hans"
	case len(n) > 2 && (n[2] == '-' || n[2] == '_'):
		if _, ok := texts[n[:2]]; ok {
			return n[:2]
		}
	}
	return "en"
}

// TextFor returns the texts of a language code.
func TextFor(code string) Text { return texts[LanguageFor(code)] }
