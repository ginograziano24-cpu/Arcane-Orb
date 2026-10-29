package loc

import "testing"

func TestTr(t *testing.T) {
	tb := New(map[string]string{
		"Break every brick":            "Разбей все кирпичи",
		"WAVE %d / %d":                 "ВОЛНА %d / %d",
		"%s Mastery %d":                "Мастерство: %s, ур. %d",
		"Berserker":                    "Берсерк",
		"Rampage":                      "Буйство",
		"Firestorm":                    "Огненная буря",
		"+0.5 orb damage.":             "+0.5 к урону сферы.",
		"%d / %d XP":                   "%[2]d из %[1]d XP",
		"Ascension %d unlocked!":       "Восхождение %d открыто!",
		" FALLS!":                      " ПАДАЕТ!",
		"A HEART FALLS!":               "ПАДАЕТ СЕРДЦЕ!",
		"Continue  ·  %s, Act %d":      "Продолжить  ·  %s, акт %d",
		"Buy %s (25 gold)":             "Купить: %s (25 золота)",
		"Heavy Core":                   "Тяжёлое ядро",
		"Choose your Origin":           "Выбери происхождение",
		"Every 5th hit ignites.":       "Каждый 5-й удар поджигает.",
		"PASSIVE  ·  ":                 "ПАССИВНОЕ  ·  ",
		"ADVANCE":                      "НАСТУПЛЕНИЕ",
		"Attunement":                   "Созвучие",
		"Something unrelated entirely": "Нечто",
		" +%d more":                    " и ещё %d",
		"Overdrive":                    "Форсаж",
		"   ·   seed ":                 "   ·   сид ",
	})
	tb.Hook = func(s string) string {
		if s == "The Sunken Vault" {
			return "Затонувшее хранилище"
		}
		return ""
	}
	cases := map[string]string{
		"Break every brick":              "Разбей все кирпичи",
		"WAVE 2 / 3":                     "ВОЛНА 2 / 3",
		"Berserker Mastery 4":            "Мастерство: Берсерк, ур. 4",
		"Rampage  ·  Firestorm":          "Буйство  ·  Огненная буря",
		"+0.5 orb damage.":               "+0.5 к урону сферы.",
		"+1.5 orb damage.":               "+1.5 orb damage.",
		"120 / 300 XP":                   "300 из 120 XP",
		"Ascension 3 unlocked!":          "Восхождение 3 открыто!",
		"Continue  ·  Rampage, Act 2":    "Продолжить  ·  Буйство, акт 2",
		"Buy Heavy Core (25 gold)":       "Купить: Тяжёлое ядро (25 золота)",
		"RAMPAGE":                        "БУЙСТВО",
		"Every 7th hit ignites.":         "Every 7th hit ignites.",
		"Every 5th hit ignites.":         "Каждый 5-й удар поджигает.",
		"Rompe":                          "Rompe",
		"Untranslated text":              "Untranslated text",
		"  Choose your Origin ":          "  Выбери происхождение ",
		"Heavy Core FALLS!":              "Тяжёлое ядро ПАДАЕТ!",
		"Overdrive +6 more":              "Форсаж и ещё 6",
		"PASSIVE  ·  ATTUNEMENT":         "ПАССИВНОЕ  ·  СОЗВУЧИЕ",
		"The Sunken Vault   ·   seed 42": "Затонувшее хранилище   ·   сид 42",
	}
	for in, want := range cases {
		if got := tb.Tr(in); got != want {
			t.Errorf("Tr(%q) = %q, want %q", in, got, want)
		}
	}
	// translated output is left alone when drawn again
	if got := tb.Tr("Разбей все кирпичи"); got != "Разбей все кирпичи" {
		t.Errorf("retranslated: %q", got)
	}
}

func TestBreakTokens(t *testing.T) {
	got := BreakTokens("打破每一块砖。Then go")
	want := []string{"打", "破", "每", "一", "块", "砖。", "Then", " ", "go"}
	if len(got) != len(want) {
		t.Fatalf("got %q", got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %q", got)
		}
	}
}
