package legacyimport

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

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
	reRegister   = regexp.MustCompile(`[А-ЯӨҮЁA-Z]{1,2}\s*\d{8,10}|\b\d{7,12}\b`)
	reOwnerSplit = regexp.MustCompile(`[,;\n]+`)
)

// splitOwners — нэр, регистрийг хос болгоно. Тоо зөрвөл БУРУУ ХҮНТЭЙ холбохын
// оронд алдаа буцаана.
func splitOwners(fullName, registerText string) ([]owner, error) {
	var registers []string
	for _, r := range reRegister.FindAllString(strings.ToUpper(registerText), -1) {
		registers = append(registers, strings.Join(strings.Fields(r), ""))
	}
	var delimited []string
	for _, part := range reOwnerSplit.Split(fullName, -1) {
		if n := normalize(part); n != "" {
			delimited = append(delimited, n)
		}
	}
	tokens := strings.Fields(normalize(fullName))
	names := delimited
	if len(registers) > 1 && len(delimited) != len(registers) {
		if len(tokens) != len(registers)*2 {
			return nil, fmt.Errorf("өмчлөгчийн нэр (%d үг), регистр (%d) зөрүүтэй: мөрийг шалгана уу", len(tokens), len(registers))
		}
		names = nil
		for i := range registers {
			names = append(names, strings.Join(tokens[i*2:i*2+2], " "))
		}
	}
	if len(registers) > 0 && len(names) != len(registers) {
		return nil, errors.New("өмчлөгчийн нэр ба регистрийн тоо зөрүүтэй")
	}
	if len(names) == 0 && len(registers) > 0 {
		return nil, errors.New("регистртэй боловч өмчлөгчийн нэр хоосон")
	}
	out := make([]owner, 0, len(names))
	for i, full := range names {
		last, first := splitPersonName(full)
		o := owner{full: full, lastName: last, name: first}
		if i < len(registers) {
			o.registerNo = registers[i]
		}
		out = append(out, o)
	}
	return out, nil
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
