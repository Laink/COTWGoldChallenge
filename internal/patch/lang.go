package patch

import "strings"

// Text holds the words shown in the game, in one language.
type Text struct {
	Title          string
	DiamondSure    string
	GoldSure       string
	GoldPossible   string // followed by a percentage
	GoldImpossible string
	NoTrophy       string
	Diamond        string // followed by a percentage
	SilverSure     string
	Silver         string // followed by a percentage
	BronzeAtBest   string
	Tiers          [4]string // bronze, silver, gold, diamond
	TiersShort     [4]string
	Percent        string // "%" or " %"
	Decimal        string // "." or ","
	SkillMissing   string // followed by the skill name (Spotting Knowledge, level 3)
}

// SkillName is the skill needed for the gauge, when the game's own name cannot be read.
const SkillName = "Spotting Knowledge 3"

var texts = map[string]Text{
	"en": {"POTENTIAL", "DIAMOND GUARANTEED", "GOLD GUARANTEED", "GOLD POSSIBLE ", "GOLD IMPOSSIBLE", "NO TROPHY",
		"diamond ", "silver guaranteed", "silver ", "bronze at best",
		[4]string{"BRONZE", "SILVER", "GOLD", "DIAMOND"}, [4]string{"BR.", "SIL.", "GOLD", "DIA."}, "%", ".", "Skill missing: "},
	"fr": {"POTENTIEL", "DIAMANT GARANTI", "OR GARANTI", "OR POSSIBLE ", "OR IMPOSSIBLE", "PAS DE TROPHÉE",
		"diamant ", "argent garanti", "argent ", "bronze au mieux",
		[4]string{"BRONZE", "ARGENT", "OR", "DIAMANT"}, [4]string{"BR.", "ARG.", "OR", "DIA."}, " %", ",", "Compétence absente : "},
	"de": {"POTENZIAL", "DIAMANT SICHER", "GOLD SICHER", "GOLD MÖGLICH ", "GOLD UNMÖGLICH", "KEINE TROPHÄE",
		"Diamant ", "Silber sicher", "Silber ", "höchstens Bronze",
		[4]string{"BRONZE", "SILBER", "GOLD", "DIAMANT"}, [4]string{"BR.", "SILB.", "GOLD", "DIA."}, " %", ",", "Fertigkeit fehlt: "},
	"es": {"POTENCIAL", "DIAMANTE ASEGURADO", "ORO ASEGURADO", "ORO POSIBLE ", "ORO IMPOSIBLE", "SIN TROFEO",
		"diamante ", "plata asegurada", "plata ", "bronce como máximo",
		[4]string{"BRONCE", "PLATA", "ORO", "DIAMANTE"}, [4]string{"BR.", "PLATA", "ORO", "DIAM."}, " %", ",", "Habilidad ausente: "},
	"it": {"POTENZIALE", "DIAMANTE GARANTITO", "ORO GARANTITO", "ORO POSSIBILE ", "ORO IMPOSSIBILE", "NESSUN TROFEO",
		"diamante ", "argento garantito", "argento ", "al massimo bronzo",
		[4]string{"BRONZO", "ARGENTO", "ORO", "DIAMANTE"}, [4]string{"BR.", "ARG.", "ORO", "DIAM."}, "%", ",", "Abilità assente: "},
	"pl": {"POTENCJAŁ", "DIAMENT PEWNY", "ZŁOTO PEWNE", "ZŁOTO MOŻLIWE ", "ZŁOTO NIEMOŻLIWE", "BRAK TROFEUM",
		"diament ", "srebro pewne", "srebro ", "najwyżej brąz",
		[4]string{"BRĄZ", "SREBRO", "ZŁOTO", "DIAMENT"}, [4]string{"BR.", "SREB.", "ZŁOTO", "DIAM."}, "%", ",", "Brak umiejętności: "},
	"ru": {"ПОТЕНЦИАЛ", "АЛМАЗ ГАРАНТИРОВАН", "ЗОЛОТО ГАРАНТИРОВАНО", "ЗОЛОТО ВОЗМОЖНО ", "ЗОЛОТО НЕВОЗМОЖНО", "НЕТ ТРОФЕЯ",
		"алмаз ", "серебро гарантировано", "серебро ", "не выше бронзы",
		[4]string{"БРОНЗА", "СЕРЕБРО", "ЗОЛОТО", "АЛМАЗ"}, [4]string{"БР.", "СЕР.", "ЗОЛ.", "АЛМ."}, "%", ",", "Нет навыка: "},
	"pt": {"POTENCIAL", "DIAMANTE GARANTIDO", "OURO GARANTIDO", "OURO POSSÍVEL ", "OURO IMPOSSÍVEL", "SEM TROFÉU",
		"diamante ", "prata garantida", "prata ", "no máximo bronze",
		[4]string{"BRONZE", "PRATA", "OURO", "DIAMANTE"}, [4]string{"BR.", "PRATA", "OURO", "DIAM."}, "%", ",", "Habilidade ausente: "},
	"cs": {"POTENCIÁL", "DIAMANT JISTÝ", "ZLATO JISTÉ", "ZLATO MOŽNÉ ", "ZLATO NEMOŽNÉ", "BEZ TROFEJE",
		"diamant ", "stříbro jisté", "stříbro ", "nejvýše bronz",
		[4]string{"BRONZ", "STŘÍBRO", "ZLATO", "DIAMANT"}, [4]string{"BR.", "STŘ.", "ZLATO", "DIA."}, " %", ",", "Chybí dovednost: "},
	"ja": {"ポテンシャル", "ダイヤ確定", "ゴールド確定", "ゴールド可能 ", "ゴールド不可", "トロフィーなし",
		"ダイヤ ", "シルバー確定", "シルバー ", "最高でブロンズ",
		[4]string{"ブロンズ", "シルバー", "ゴールド", "ダイヤ"}, [4]string{"ブロンズ", "シルバー", "ゴールド", "ダイヤ"}, "%", ".", "スキル未習得："},
	"zh-hans": {"潜力", "必定钻石", "必定金牌", "可能金牌 ", "不可能金牌", "无奖杯",
		"钻石 ", "必定银牌", "银牌 ", "最多铜牌",
		[4]string{"铜牌", "银牌", "金牌", "钻石"}, [4]string{"铜", "银", "金", "钻"}, "%", ".", "缺少技能："},
	"zh-hant": {"潛力", "必定鑽石", "必定金牌", "可能金牌 ", "不可能金牌", "無獎盃",
		"鑽石 ", "必定銀牌", "銀牌 ", "最多銅牌",
		[4]string{"銅牌", "銀牌", "金牌", "鑽石"}, [4]string{"銅", "銀", "金", "鑽"}, "%", ".", "缺少技能："},
	"ko": {"잠재력", "다이아몬드 확정", "골드 확정", "골드 가능 ", "골드 불가", "트로피 없음",
		"다이아몬드 ", "실버 확정", "실버 ", "최대 브론즈",
		[4]string{"브론즈", "실버", "골드", "다이아몬드"}, [4]string{"브론즈", "실버", "골드", "다이아"}, "%", ".", "스킬 없음: "},
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
	return "en"
}

// TextFor returns the texts of a language code.
func TextFor(code string) Text { return texts[LanguageFor(code)] }

// width estimates the rendered width of a label at size 13.
func width(s string) int {
	w := 0
	for _, r := range s {
		if r >= 0x1100 {
			w += 13
		} else {
			w += 8
		}
	}
	return w
}
