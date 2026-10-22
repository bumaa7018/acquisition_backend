package legacyimport

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"acquisition_backend/internal/domain"
)

// ── Нэгж талбарын төлөв ─────────────────────────────────────────────────────

// Системийн parcel_status.id (= code) — зөвхөн эксэлийн тайлбараас таамаглахад.
const (
	statusProcessing  = domain.ParcelStatusProcessing
	statusEvaluating  = domain.ParcelStatusEvaluating
	statusAffectedOut = domain.ParcelStatusAffectedOut
	statusRejected    = domain.ParcelStatusRejected
	statusReleased    = domain.ParcelStatusReleased
)

// statusCatalog — системийн нэгж талбарын төлөвүүд (parcel_status). SHP-ийн
// төлөвийн кодыг СИСТЕМИЙН кодтой (101, 102 … админы нэмсэн 110 …) тулгана.
// Таараагүй / кодгүй нэгж талбар хамгийн ЭХНИЙ төлөвт (sort_order) орно.
type statusCatalog struct {
	first  statusEntry
	byCode map[string]statusEntry
}

type statusEntry struct {
	id   int
	name string
}

func loadStatusCatalog(ctx context.Context, q querier) (*statusCatalog, error) {
	rows, err := q.Query(ctx, `SELECT id, TRIM(code), name FROM parcel_status ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	c := &statusCatalog{byCode: map[string]statusEntry{}}
	n := 0
	for rows.Next() {
		var e statusEntry
		var code string
		if err := rows.Scan(&e.id, &code, &e.name); err != nil {
			return nil, err
		}
		if n == 0 {
			c.first = e
		}
		n++
		c.byCode[code] = e
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("системд нэгж талбарын төлөв (parcel_status) бүртгэгдээгүй байна")
	}
	return c, nil
}

type statusMapping struct {
	status int
	// mapped — SHP-ийн код системийн төлөвтэй таарсан.
	mapped bool
	// unmatched — код байгаа ч системд таарах төлөвгүй (эхний төлөвт орно).
	unmatched bool
	label     string
}

func (c *statusCatalog) mapCode(code *int) statusMapping {
	if code == nil {
		return statusMapping{status: c.first.id, label: c.first.name}
	}
	if hit, ok := c.byCode[strconv.Itoa(*code)]; ok {
		return statusMapping{status: hit.id, mapped: true, label: hit.name}
	}
	return statusMapping{status: c.first.id, unmatched: true, label: c.first.name}
}

// initialStatusID — хамгийн эхний төлөв (импортын анхны түүх, кодгүй нэгж талбар).
func initialStatusID(ctx context.Context, q querier) (int, error) {
	var id int
	err := q.QueryRow(ctx, `SELECT id FROM parcel_status ORDER BY sort_order, id LIMIT 1`).Scan(&id)
	return id, err
}

var noteStatusRules = []struct {
	re     *regexp.Regexp
	status int
}{
	{regexp.MustCompile(`(?i)татгалз`), statusRejected},
	{regexp.MustCompile(`(?i)нөлөөлл?өөс\s*гар`), statusAffectedOut},
	{regexp.MustCompile(`(?i)чөлөөлс[өе]н`), statusReleased},
	{regexp.MustCompile(`(?i)үнэлгээ`), statusEvaluating},
	{regexp.MustCompile(`(?i)зөвшилц`), statusProcessing},
}

// statusFromNote — shp-ийн код тохируулаагүй үед эксэлийн тайлбараас төлөв.
func statusFromNote(note string) (int, bool) {
	text := strings.TrimSpace(note)
	if text == "" {
		return 0, false
	}
	for _, rule := range noteStatusRules {
		if rule.re.MatchString(text) {
			return rule.status, true
		}
	}
	return 0, false
}

var reLandGrant = regexp.MustCompile(`(?i)газар|газр`)

// compensationType — «газар/газр» агуулбал land_grant, бусад нь cash.
func compensationType(formLabel string) (kind, source string) {
	kind = "cash"
	if reLandGrant.MatchString(formLabel) {
		kind = "land_grant"
	}
	source = "default"
	if strings.TrimSpace(formLabel) != "" {
		source = "form"
	}
	return kind, source
}

// ── Өмчлөгч ─────────────────────────────────────────────────────────────────

type owner struct {
	full, lastName, name, registerNo string
}

var (
	reRegister = regexp.MustCompile(`[А-ЯӨҮЁA-Z]{1,2}\s*\d{8,10}|\b\d{7,12}\b`)
	// reOwnerSegment — «,» «;» нь ХҮН тусгаарлана (зай, мөр шилжилт нь үг).
	reOwnerSegment = regexp.MustCompile(`[,;]+`)
	// reCamelBoundary — наалдсан овог нэрийн зааг: «АнарынЦэцэг» → «Анарын|Цэцэг».
	reCamelBoundary = regexp.MustCompile(`\p{Ll}\p{Lu}`)
)

// splitOwners — нэр, регистрийг хос болгоно. Тоо зөрвөл БУРУУ ХҮНТЭЙ холбохын
// оронд алдаа буцаана.
func splitOwners(fullName, registerText string) ([]owner, error) {
	var registers []string
	for _, r := range reRegister.FindAllString(strings.ToUpper(registerText), -1) {
		registers = append(registers, strings.Join(strings.Fields(r), ""))
	}
	owners, err := parseOwnerNames(fullName)
	if err != nil {
		return nil, err
	}
	if len(owners) == 0 && len(registers) > 0 {
		return nil, errors.New("регистртэй боловч өмчлөгчийн нэр хоосон")
	}
	if len(registers) > 0 && len(owners) != len(registers) {
		return nil, fmt.Errorf("өмчлөгч %d (%s), регистр %d — тоо зөрүүтэй", len(owners), ownerNames(owners), len(registers))
	}
	for i := range owners {
		if i < len(registers) {
			owners[i].registerNo = registers[i]
		}
	}
	return owners, nil
}

// parseOwnerNames — өмчлөгчдийн овог, нэр. «,» «;» нь ХҮН тусгаарлана; хэсэг
// бүрийн дотор зай, мөр шилжилт нь ҮГ тусгаарлана («Б⏎Бат» нэг хүн), хүн
// бүр = овог + нэр:
//
//	«Б.Бат»       — цэгээр: овог «Б», нэр «Бат» («Б.» бол нэр нь дараагийн үг)
//	«АнарынЦэцэг» — жижиг→том үсгийн заагаар: овог «Анарын», нэр «Цэцэг»
//	«Б Бат»       — бусад үгс дарааллаараа хос хосоороо (овог, нэр)
//
// Хуулийн этгээд (ХХК …) агуулсан хэсэг бүтнээрээ нэг өмчлөгч. Хэсэгт ганц үг
// бол нэр (овоггүй). Хосгүй үлдсэн үг — хэн болохыг таах боломжгүй тул алдаа.
func parseOwnerNames(text string) ([]owner, error) {
	var out []owner
	push := func(last, name string) {
		out = append(out, owner{full: strings.TrimSpace(last + " " + name), lastName: last, name: name})
	}
	unpaired := func(word string) error {
		return fmt.Errorf("«%s» — овог/нэрийн хосгүй үг: өмчлөгчдийг шалгана уу", word)
	}
	for _, seg := range reOwnerSegment.Split(text, -1) {
		seg = normalize(seg)
		if seg == "" {
			continue
		}
		if reLegalEntity.MatchString(seg) {
			out = append(out, owner{full: seg, name: seg})
			continue
		}
		before := len(out)
		pending := "" // нэрээ хүлээж буй овог — хүн тусгаарлагчийг давахгүй
		for _, word := range strings.Fields(seg) {
			if pending != "" {
				// Хүлээж буй овгийн нэр — бүтэн үгээрээ («Б. БатЭрдэнэ»), гэхдээ
				// өөрөө бүтэн хос («Б А.Цэцэг») бол хэнийх нь тодорхойгүй.
				if i := strings.LastIndex(word, "."); i > 0 && i < len(word)-1 {
					return nil, unpaired(pending)
				}
				push(pending, strings.Trim(word, "."))
				pending = ""
				continue
			}
			if i := strings.LastIndex(word, "."); i >= 0 {
				last, name := word[:i], word[i+1:]
				switch {
				case last != "" && name != "":
					push(last, name)
				case last != "":
					pending = last
				case name != "":
					pending = name
				}
				continue
			}
			if loc := reCamelBoundary.FindStringIndex(word); loc != nil {
				// Зааг = эхний жижиг үсгийн дараа (кирилл үсэг олон байт).
				_, size := utf8.DecodeRuneInString(word[loc[0]:])
				cut := loc[0] + size
				push(word[:cut], word[cut:])
				continue
			}
			pending = word
		}
		if pending != "" {
			if len(out) > before {
				return nil, unpaired(pending)
			}
			push("", pending) // «Саранчимэг» — овоггүй нэг хүн
		}
	}
	return out, nil
}

// ownerNames — «Б Бат · А Цэцэг» (анхааруулгын текстэд).
func ownerNames(owners []owner) string {
	names := make([]string, len(owners))
	for i, o := range owners {
		names[i] = o.full
	}
	return strings.Join(names, " · ")
}

// ownerIssues — өмчлөгчийн овог нэр/регистр дутуу эсвэл зөрүүтэйг шалгалтын
// шатанд анхааруулна (эс бөгөөс зөвхөн «Оруулах»-ын логт гардаг байв).
// Мөр бүртгэгдэнэ — зөвхөн эзэмшигч нь дутуу/огт орохгүй.
func ownerIssues(label string, row *ReportRow) []Issue {
	name, reg := normalize(row.HolderName), normalize(row.HolderRegisterNo)
	switch {
	case name == "" && reg == "":
		return []Issue{warnIssue("owner_missing", label+": өмчлөгчийн овог нэр, регистр хоосон — эзэмшигч бүртгэгдэхгүй")}
	case name == "":
		return []Issue{warnIssue("owner_missing", fmt.Sprintf("%s: өмчлөгчийн овог нэр хоосон (регистр %s) — эзэмшигч бүртгэгдэхгүй", label, reg))}
	}
	owners, err := splitOwners(row.HolderName, row.HolderRegisterNo)
	if err != nil {
		return []Issue{warnIssue("owner_mismatch", fmt.Sprintf("%s: «%s» / регистр «%s» — %s — эзэмшигч бүртгэгдэхгүй", label, name, reg, err.Error()))}
	}
	var out []Issue
	if len(owners) > 1 {
		// Задалсан үр дүнг харуулна — хүмүүс зөв салсныг нүдээр шалгана.
		parts := make([]string, len(owners))
		for i, o := range owners {
			parts[i] = fmt.Sprintf("овог «%s» нэр «%s»", o.lastName, o.name)
			if o.registerNo != "" {
				parts[i] += " " + o.registerNo
			}
		}
		out = append(out, infoIssue("owners_parsed", fmt.Sprintf("%s: %d өмчлөгч — %s", label, len(owners), strings.Join(parts, "; "))))
	}
	if reg == "" {
		out = append(out, warnIssue("owner_no_register", fmt.Sprintf("%s: «%s» регистргүй — эзэмшигч регистргүй бүртгэгдэнэ", label, name)))
	}
	return out
}

func ownerCaution(n int) string {
	return fmt.Sprintf("%d мөрийн өмчлөгчийн овог нэр/регистр дутуу эсвэл зөрүүтэй — эзэмшигч бүртгэгдэхгүй эсвэл регистргүй орно (үнэлгээний алхмын «Өмчлөгч» шүүлтүүрээс харна уу)", n)
}

// countOwnerIssues — өмчлөгчийн анхааруулгатай мөрийн тоо.
func countOwnerIssues(issues []Issue) int {
	if hasCode(issues, "owner_missing") || hasCode(issues, "owner_mismatch") || hasCode(issues, "owner_no_register") {
		return 1
	}
	return 0
}

// ── Хаяг ────────────────────────────────────────────────────────────────────

type address struct {
	district, au3, street, khashaa, raw string
}

var (
	reDistrict = regexp.MustCompile(`(?i)([\p{L}-]+)\s+(?:дүүр(?:гийн|эг)|сум(?:ын)?)`)
	reAU3      = regexp.MustCompile(`(?i)(\d+)\s*(?:дугаар|дүгээр|-р|р)?\s*(?:хороо|баг)`)
	reStreet1  = regexp.MustCompile(`(?i)(?:хороо(?:ны)?|баг(?:ийн)?)\s+(.+?)\s+(?:гудамж(?:ны)?|гудамж|өргөн чөлөө)`)
	reStreet2  = regexp.MustCompile(`(?i)(?:^|,\s*)([^,]+?)\s+гудамж(?:ны)?`)
	reKhashaa  = regexp.MustCompile(`(?i)([\p{L}\d/-]+)\s*тоот`)
)

func group1(re *regexp.Regexp, text string) string {
	if m := re.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return ""
}

func parseAddress(value string) address {
	text := normalize(value)
	street := group1(reStreet1, text)
	if street == "" {
		street = group1(reStreet2, text)
	}
	return address{
		district: group1(reDistrict, text),
		au3:      group1(reAU3, text),
		street:   strings.TrimSpace(street),
		khashaa:  group1(reKhashaa, text),
		raw:      text,
	}
}

// ── Захирамж ────────────────────────────────────────────────────────────────

type decreeRef struct {
	number string
	date   string // YYYY-MM-DD эсвэл хоосон
}

var (
	reDecreeLetter = regexp.MustCompile(`[AА]\s*[-/]?\s*(\d+)`)
	reDecreeNo     = regexp.MustCompile(`[АБ]\s*/\s*\d+`)
	reDecreeWord   = regexp.MustCompile(`(?:№|ДУГААР)\s*(\d+(?:/\d+)?)`)
	reDecreeBare   = regexp.MustCompile(`^\d+(?:/\d+)?$`)
	reHasDigit     = regexp.MustCompile(`\d`)
)

// parseDecree — "2025.01.24-А70" → {А/70, 2025-01-24}. Тайлбар бичигдсэн нүд
// (дугааргүй) бол алдаа; хоосон бол nil.
func parseDecree(text string) (*decreeRef, error) {
	normalized := reDecreeLetter.ReplaceAllString(strings.ToUpper(normalize(text)), "А/${1}")
	seen := map[string]bool{}
	var matches []string
	for _, m := range reDecreeNo.FindAllString(normalized, -1) {
		if !seen[m] {
			seen[m] = true
			matches = append(matches, m)
		}
	}
	if len(matches) > 1 {
		return nil, errors.New("нэг мөрөнд олон захирамжийн дугаар байна: холбоосыг ялгана уу")
	}
	number := ""
	switch {
	case len(matches) == 1:
		number = strings.Join(strings.Fields(matches[0]), "")
	case reDecreeWord.MatchString(normalized):
		number = reDecreeWord.FindStringSubmatch(normalized)[1]
	case reDecreeBare.MatchString(normalized):
		number = normalized
	}
	if number == "" {
		if normalized == "" || !reHasDigit.MatchString(normalized) {
			return nil, nil
		}
		return nil, fmt.Errorf("захирамжийн дугаар танигдсангүй: %s", text)
	}
	return &decreeRef{number: number, date: parseDate(text)}, nil
}

// ── Дүн тэнцүүлэх ───────────────────────────────────────────────────────────

type balancedAmounts struct {
	landAmount       float64
	adjustment       float64
	breakdownMissing bool
	residual         float64
}

// balanceAmounts — «Нийт нөхөх олговор»-оос ДУТУУ задаргааг газрын үнэ дээр
// нөхнө. Задаргаа нийт дүнгээс ИХ бол бууруулахгүй, residual-аар мэдээлнэ.
func balanceAmounts(r *ReportRow) balancedAmounts {
	assets := r.RealStateAmount + r.PropertyAmount
	out := balancedAmounts{
		landAmount:       r.LandAmount,
		breakdownMissing: r.LandAmount <= 0 && assets <= 0 && r.TotalAmount > 0,
	}
	if r.TotalAmount <= 0 {
		return out
	}
	diff := r.TotalAmount - (r.LandAmount + assets)
	if math.Abs(diff) <= 0.01 {
		return out
	}
	if diff < 0 {
		out.residual = diff
		return out
	}
	out.landAmount = r.LandAmount + diff
	out.adjustment = diff
	return out
}

var (
	reAssetLetter = regexp.MustCompile(`[ҮYV]\s*[-–]?\s*(\d+)`)
	reAssetNo     = regexp.MustCompile(`Ү-\d+`)
	reAssetSplit  = regexp.MustCompile(`[,;\s/]+`)
)

// assetNumbers — «Ү-дугаар» нүднээс хөрөнгийн дугаарууд.
func assetNumbers(text string) []string {
	normalized := reAssetLetter.ReplaceAllString(strings.ToUpper(text), "Ү-${1}")
	seen := map[string]bool{}
	var out []string
	add := func(v string) {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	if ids := reAssetNo.FindAllString(normalized, -1); len(ids) > 0 {
		for _, id := range ids {
			add(id)
		}
		return out
	}
	for _, v := range reAssetSplit.Split(normalize(text), -1) {
		if reDigits6.MatchString(v) {
			add(v)
		}
	}
	return out
}
