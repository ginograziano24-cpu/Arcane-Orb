package main

import "strings"

// Chamber names are built from two word lists ("The Sunken Reliquary"), so
// each language puts them together with its own grammar: word order,
// articles and adjective agreement. Index i matches levelgen's lists.

var nameAdjEN = []string{
	"Whispering", "Ember", "Frozen", "Gilded", "Hollow", "Shattered", "Moonlit",
	"Thorned", "Sunken", "Crimson", "Verdant", "Starfallen", "Howling", "Silent",
	"Obsidian", "Forgotten", "Glimmering", "Twisted", "Ancient", "Drifting",
	"Burning", "Azure", "Haunted", "Radiant", "Crumbling", "Veiled", "Humming", "Fractured",
}
var nameNounEN = []string{
	"Vault", "Spire", "Labyrinth", "Sanctum", "Gate", "Grove", "Citadel",
	"Crypt", "Observatory", "Bastion", "Archive", "Forge", "Cloister",
	"Reliquary", "Keep", "Hollows", "Altar", "Library", "Garden",
	"Catacombs", "Tower", "Bridge", "Caverns", "Hall", "Orrery", "Engine", "Furnace",
}

// zh / ja: adjective then noun, no agreement
var nameAdjZH = strings.Split("低语|余烬|冰封|镀金|空洞|破碎|月光|荆棘|沉没|深红|翠绿|星陨|嚎风|寂静|黑曜|遗忘|微光|扭曲|远古|漂流|燃烧|蔚蓝|闹鬼|光辉|崩塌|蒙纱|嗡鸣|断裂", "|")
var nameNounZH = strings.Split("宝库|尖塔|迷宫|圣所|大门|林地|城堡|地穴|观星台|堡垒|档案馆|熔炉|回廊|圣物堂|要塞|空谷|祭坛|图书馆|花园|地下墓穴|高塔|长桥|洞窟|大厅|天体仪|引擎|熔窑", "|")
var nameAdjJA = strings.Split("囁く|残り火の|凍てついた|金色の|虚ろな|砕けた|月明かりの|茨の|沈んだ|深紅の|翠緑の|星降る|吠える|静かなる|黒曜の|忘れられた|煌めく|歪んだ|古の|漂う|燃える|蒼き|呪われた|輝く|崩れゆく|覆われた|唸る|ひび割れた", "|")
var nameNounJA = strings.Split("宝物庫|尖塔|迷宮|聖域|門|森|城塞|地下聖堂|天文台|砦|書庫|鍛冶場|回廊|聖遺物堂|城|窪地|祭壇|図書館|庭園|地下墓地|塔|橋|洞窟|広間|天球儀|機関|炉", "|")

// ru: adjective forms m|f|n|pl; nouns "word:gender"
var nameAdjRU = []string{
	"Шепчущий|Шепчущая|Шепчущее|Шепчущие", "Тлеющий|Тлеющая|Тлеющее|Тлеющие", "Застывший|Застывшая|Застывшее|Застывшие",
	"Позолоченный|Позолоченная|Позолоченное|Позолоченные", "Пустой|Пустая|Пустое|Пустые", "Разбитый|Разбитая|Разбитое|Разбитые",
	"Лунный|Лунная|Лунное|Лунные", "Тернистый|Тернистая|Тернистое|Тернистые", "Затонувший|Затонувшая|Затонувшее|Затонувшие",
	"Багровый|Багровая|Багровое|Багровые", "Зелёный|Зелёная|Зелёное|Зелёные", "Звездопадный|Звездопадная|Звездопадное|Звездопадные",
	"Воющий|Воющая|Воющее|Воющие", "Безмолвный|Безмолвная|Безмолвное|Безмолвные", "Обсидиановый|Обсидиановая|Обсидиановое|Обсидиановые",
	"Забытый|Забытая|Забытое|Забытые", "Мерцающий|Мерцающая|Мерцающее|Мерцающие", "Искажённый|Искажённая|Искажённое|Искажённые",
	"Древний|Древняя|Древнее|Древние", "Дрейфующий|Дрейфующая|Дрейфующее|Дрейфующие", "Горящий|Горящая|Горящее|Горящие",
	"Лазурный|Лазурная|Лазурное|Лазурные", "Проклятый|Проклятая|Проклятое|Проклятые", "Сияющий|Сияющая|Сияющее|Сияющие",
	"Рушащийся|Рушащаяся|Рушащееся|Рушащиеся", "Сокрытый|Сокрытая|Сокрытое|Сокрытые", "Гудящий|Гудящая|Гудящее|Гудящие",
	"Расколотый|Расколотая|Расколотое|Расколотые",
}
var nameNounRU = strings.Split("Хранилище:n|Шпиль:m|Лабиринт:m|Святилище:n|Врата:p|Роща:f|Цитадель:f|Склеп:m|Обсерватория:f|Бастион:m|Архив:m|Кузня:f|Обитель:f|Реликварий:m|Твердыня:f|Лощины:p|Алтарь:m|Библиотека:f|Сад:m|Катакомбы:p|Башня:f|Мост:m|Пещеры:p|Зал:m|Планетарий:m|Механизм:m|Печь:f", "|")

// de: weak adjective stems after the definite article (+e, plural +en)
var nameAdjDE = strings.Split("Flüsternd|Glühend|Gefroren|Vergoldet|Hohl|Zerschmettert|Mondbeschienen|Dornig|Versunken|Karmesinrot|Grünend|Sternfunkelnd|Heulend|Still|Obsidianschwarz|Vergessen|Schimmernd|Verdreht|Uralt|Treibend|Brennend|Azurblau|Verwunschen|Strahlend|Bröckelnd|Verschleiert|Summend|Geborsten", "|")
var nameNounDE = strings.Split("Gewölbe:n|Turmspitze:f|Labyrinth:n|Heiligtum:n|Tor:n|Hain:m|Zitadelle:f|Gruft:f|Sternwarte:f|Bastion:f|Archiv:n|Schmiede:f|Kreuzgang:m|Reliquienschrein:m|Burgfried:m|Senken:p|Altar:m|Bibliothek:f|Garten:m|Katakomben:p|Turm:m|Brücke:f|Höhlen:p|Halle:f|Planetarium:n|Maschine:f|Hochofen:m", "|")

// es / pt: article, noun, then the adjective agreeing (m|f|m-pl|f-pl);
// genders m, f, p (masc. plural), P (fem. plural)
var nameAdjES = []string{
	"Susurrante|Susurrante|Susurrantes|Susurrantes", "de Ascuas|de Ascuas|de Ascuas|de Ascuas", "Helado|Helada|Helados|Heladas",
	"Dorado|Dorada|Dorados|Doradas", "Hueco|Hueca|Huecos|Huecas", "Destrozado|Destrozada|Destrozados|Destrozadas",
	"Lunar|Lunar|Lunares|Lunares", "Espinoso|Espinosa|Espinosos|Espinosas", "Hundido|Hundida|Hundidos|Hundidas",
	"Carmesí|Carmesí|Carmesíes|Carmesíes", "Verde|Verde|Verdes|Verdes", "de las Estrellas Caídas|de las Estrellas Caídas|de las Estrellas Caídas|de las Estrellas Caídas",
	"Aullante|Aullante|Aullantes|Aullantes", "Silencioso|Silenciosa|Silenciosos|Silenciosas", "de Obsidiana|de Obsidiana|de Obsidiana|de Obsidiana",
	"Olvidado|Olvidada|Olvidados|Olvidadas", "Centelleante|Centelleante|Centelleantes|Centelleantes", "Retorcido|Retorcida|Retorcidos|Retorcidas",
	"Antiguo|Antigua|Antiguos|Antiguas", "Errante|Errante|Errantes|Errantes", "Ardiente|Ardiente|Ardientes|Ardientes",
	"Azur|Azur|Azures|Azures", "Encantado|Encantada|Encantados|Encantadas", "Radiante|Radiante|Radiantes|Radiantes",
	"Ruinoso|Ruinosa|Ruinosos|Ruinosas", "Velado|Velada|Velados|Veladas", "Zumbante|Zumbante|Zumbantes|Zumbantes",
	"Fracturado|Fracturada|Fracturados|Fracturadas",
}
var nameNounES = strings.Split("Bóveda:f|Aguja:f|Laberinto:m|Sagrario:m|Puerta:f|Arboleda:f|Ciudadela:f|Cripta:f|Observatorio:m|Bastión:m|Archivo:m|Forja:f|Claustro:m|Relicario:m|Torreón:m|Hondonadas:P|Altar:m|Biblioteca:f|Jardín:m|Catacumbas:P|Torre:f|Puente:m|Cavernas:P|Salón:m|Planetario:m|Motor:m|Horno:m", "|")
var nameAdjPT = []string{
	"Sussurrante|Sussurrante|Sussurrantes|Sussurrantes", "das Brasas|das Brasas|das Brasas|das Brasas", "Congelado|Congelada|Congelados|Congeladas",
	"Dourado|Dourada|Dourados|Douradas", "Oco|Oca|Ocos|Ocas", "Despedaçado|Despedaçada|Despedaçados|Despedaçadas",
	"Enluarado|Enluarada|Enluarados|Enluaradas", "Espinhoso|Espinhosa|Espinhosos|Espinhosas", "Submerso|Submersa|Submersos|Submersas",
	"Carmesim|Carmesim|Carmesins|Carmesins", "Verdejante|Verdejante|Verdejantes|Verdejantes", "das Estrelas Cadentes|das Estrelas Cadentes|das Estrelas Cadentes|das Estrelas Cadentes",
	"Uivante|Uivante|Uivantes|Uivantes", "Silencioso|Silenciosa|Silenciosos|Silenciosas", "de Obsidiana|de Obsidiana|de Obsidiana|de Obsidiana",
	"Esquecido|Esquecida|Esquecidos|Esquecidas", "Cintilante|Cintilante|Cintilantes|Cintilantes", "Retorcido|Retorcida|Retorcidos|Retorcidas",
	"Antigo|Antiga|Antigos|Antigas", "Errante|Errante|Errantes|Errantes", "Ardente|Ardente|Ardentes|Ardentes",
	"Azul|Azul|Azuis|Azuis", "Assombrado|Assombrada|Assombrados|Assombradas", "Radiante|Radiante|Radiantes|Radiantes",
	"em Ruínas|em Ruínas|em Ruínas|em Ruínas", "Velado|Velada|Velados|Veladas", "Zumbente|Zumbente|Zumbentes|Zumbentes",
	"Fraturado|Fraturada|Fraturados|Fraturadas",
}
var nameNounPT = strings.Split("Cofre:m|Agulha:f|Labirinto:m|Sacrário:m|Portão:m|Bosque:m|Cidadela:f|Cripta:f|Observatório:m|Bastião:m|Arquivo:m|Forja:f|Claustro:m|Relicário:m|Torreão:m|Baixadas:P|Altar:m|Biblioteca:f|Jardim:m|Catacumbas:P|Torre:f|Ponte:f|Cavernas:P|Salão:m|Planetário:m|Motor:m|Fornalha:f", "|")

func nameIndex(list []string, w string) int {
	for i, s := range list {
		if s == w {
			return i
		}
	}
	return -1
}

// chamberName translates a generated chamber name; "" if s isn't one.
func chamberName(s string) string {
	if curLang == "en" || !strings.HasPrefix(s, "The ") {
		return ""
	}
	parts := strings.SplitN(s[4:], " ", 2)
	if len(parts) != 2 {
		return ""
	}
	a, n := nameIndex(nameAdjEN, parts[0]), nameIndex(nameNounEN, parts[1])
	if a < 0 || n < 0 {
		return ""
	}
	split := func(entry string) (string, string) {
		w, g, _ := strings.Cut(entry, ":")
		return w, g
	}
	switch curLang {
	case "zh":
		return nameAdjZH[a] + "之" + nameNounZH[n]
	case "ja":
		return nameAdjJA[a] + nameNounJA[n]
	case "ru":
		w, g := split(nameNounRU[n])
		forms := strings.Split(nameAdjRU[a], "|")
		return forms[strings.Index("mfnp", g)] + " " + strings.ToLower(w[:2]) + w[2:]
	case "de":
		w, g := split(nameNounDE[n])
		art := map[string]string{"m": "Der", "f": "Die", "n": "Das", "p": "Die"}[g]
		end := "e"
		if g == "p" {
			end = "en"
		}
		return art + " " + nameAdjDE[a] + end + " " + w
	case "es", "pt":
		nouns, adjs := nameNounES, nameAdjES
		arts := map[string]string{"m": "El", "f": "La", "p": "Los", "P": "Las"}
		if curLang == "pt" {
			nouns, adjs = nameNounPT, nameAdjPT
			arts = map[string]string{"m": "O", "f": "A", "p": "Os", "P": "As"}
		}
		w, g := split(nouns[n])
		forms := strings.Split(adjs[a], "|")
		return arts[g] + " " + w + " " + forms[strings.Index("mfpP", g)]
	}
	return ""
}
