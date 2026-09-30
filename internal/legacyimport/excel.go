package legacyimport

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ── Үнэлгээний тайлангийн багана ────────────────────────────────────────────

type columnSpec struct {
	key     string
	label   string
	aliases []string
}

// columns — дараалал нь чухал: толгой тулгахад ЭХЭЛЖ таарсан багана авна.
var columns = []columnSpec{
	{"rowNo", "№", []string{"№", "N", "д/д"}},
	{"constructionType", "Бүтээн байгуулалтын төрөл", []string{"Бүтээн байгуулалтын төрөл"}},
	{"projectName", "Бүтээн байгуулалтын ажлын нэр", []string{"Бүтээн байгуулалтын ажлын нэр"}},
	{"decisionNo", "НИТХ тогтоолын дугаар", []string{"НИТХ тогтоолын дугаар", "Үндэслэл"}},
	{"listNo", "Бүлэг жагсаалын дугаар", []string{"Бүлэг жагсаалын дугаар"}},
	{"fundingSource", "Хөрөнгийн эх үүсвэр", []string{"Хөрөнгийн эх үүсвэр"}},
	{"decree", "Нөхөх олговор олгосон НЗД-ын захирамж", []string{"Нөхөх олговор олгосон", "Нөхөх олговор олгосон НЗД-ын захирамжийн огноо дугаар"}},
	{"holderName", "Овог нэр", []string{"Овог нэр", "Газар өмчлөгч", "Газар өмчлөгч, эзэмшигч, ашиглагчийн нэр"}},
	{"holderRegisterNo", "Регистрийн дугаар", []string{"Регистрийн дугаар", "Газар өмчлөгч, эзэмшигчийн регистрийн дугаар"}},
	{"address", "Хаяг", []string{"Хаяг"}},
	{"parcelID", "Нэгж талбарын дугаар", []string{"Нэгж талбарын дугаар"}},
	{"areaM2", "Үндсэн талбайн хэмжээ /м2/", []string{"Үндсэн талбайн хэмжээ"}},
	{"rightType", "Эдэлбэрийн хэлбэр", []string{"Эдэлбэрийн хэлбэр"}},
	{"certificateNo", "Гэрчилгээний дугаар", []string{"Гэрчилгээний дугаар"}},
	{"affectedAreaM2", "Нөлөөлөлд өртсөн газрын хэмжээ /м2/", []string{"Хэмжээ /м2/", "Хэмжээ"}},
	{"landAmount", "Нөлөөлөлд өртсөн газрын үнэ /төгрөг/", []string{"Үнэ /төгрөг/", "Үнэ"}},
	{"assetNumber", "Ү-дугаар", []string{"Ү- дугаар", "Ү-дугаар", "Үдугаар", "Гэрчилгээний дугаар"}},
	{"realStateAmount", "Үл хөдлөх хөрөнгийн үнэ /төгрөг/", []string{"Үл хөдлөх хөрөнгө", "Үнэ /төгрөг/", "Үнэ"}},
	{"propertyAmount", "Эд хөрөнгө", []string{"Эд хөрөнгө", "Эд хөрөнгө, бусад"}},
	{"totalAmount", "Нийт нөхөх олговор /төгрөг/", []string{"Нийт нөхөх олговор", "Нийт нөхөх олговор /төгрөг/"}},
	{"compensationType", "Нөхөх олговрын хэлбэр", []string{"Нөхөх олговрын хэлбэр"}},
	{"dbChangeNote", "Мэдээллийн санд өөрчлөлт орсон эсхүл устгагдсан эсэх", []string{"Мэдээллийн санд өөрчлөлт", "Мэдээллийн санд өөрчлөлт орсон эсхүл устгагдсан эсэх"}},
	{"changedParcelID", "Өөрчлөгдсөн нэгж талбарын дугаар", []string{"Өөрчлөгдсөн нэгж талбарын дугаар"}},
	{"changedAreaM2", "Талбайн хэмжээ", []string{"Талбайн хэмжээ"}},
	{"note", "Нэмэлт тайлбар", []string{"Нэмэлт тайлбар", "Тайлбар", "Шалтгаан", "Тодорхойлолт", "Тэмдэглэл"}},
}

// requiredColumns — тайлангийн загвар таарсан гэж үзэх үндсэн 20 багана.
var requiredColumns = []string{
	"projectName", "decisionNo", "listNo", "fundingSource", "decree", "holderName",
	"holderRegisterNo", "address", "parcelID", "areaM2", "rightType", "certificateNo",
	"affectedAreaM2", "landAmount", "assetNumber", "realStateAmount", "propertyAmount",
	"totalAmount", "compensationType", "dbChangeNote",
}

var columnAliasKeys = func() map[string][]string {
	out := map[string][]string{}
	for _, c := range columns {
		for _, a := range append([]string{c.label}, c.aliases...) {
			if k := headerKey(a); k != "" {
				out[c.key] = append(out[c.key], k)
			}
		}
	}
	return out
}()

func matchesColumn(key, headerText string) bool {
	target := headerKey(headerText)
	if target == "" {
		return false
	}
	for _, alias := range columnAliasKeys[key] {
		if alias == target {
			return true
		}
	}
	return false
}

var (
	skipSheets = map[string]bool{"нийт": true, "дүн": true, "нийт дүн": true}
	reYear     = regexp.MustCompile(`^\s*(\d{4})\s*$`)
)

// emptyRowLimit — хоосон хэлбэржүүлэлтээс болж сая гаруй мөр мэдээлдэг sheet-ийг
// дуустал гүйхгүй.
const emptyRowLimit = 200

// sheetTable — нэг sheet-ийн форматласан ба түүхий утга (1-ээс эхэлсэн индекс).
type sheetTable struct{ text, raw [][]string }

func (t sheetTable) cell(rows [][]string, r, c int) string {
	if r < 1 || r > len(rows) || c < 1 || c > len(rows[r-1]) {
		return ""
	}
	return rows[r-1][c-1]
}

// readReport — эксэлийн БҮХ sheet-ээс загвартай таарсныг уншина. notes нь
// алгасагдсан sheet/мөрийн тайлбар (анхааруулга).
func readReport(name string, data []byte) ([]ReportRow, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("эксэл файлыг нээж чадсангүй: %w", err)
	}
	defer f.Close()

	var rows []ReportRow
	var notes []string
	matched := 0
	for _, sheet := range f.GetSheetList() {
		if skipSheets[strings.ToLower(normalize(sheet))] {
			continue
		}
		text, err1 := f.GetRows(sheet)
		raw, err2 := f.GetRows(sheet, excelize.Options{RawCellValue: true})
		if err1 != nil || err2 != nil {
			notes = append(notes, fmt.Sprintf("%s / %s: sheet уншигдсангүй", name, sheet))
			continue
		}
		parsed, sheetNotes, ok := readReportSheet(name, sheet, sheetTable{text: text, raw: raw})
		notes = append(notes, sheetNotes...)
		if !ok {
			continue
		}
		matched++
		rows = append(rows, parsed...)
	}
	if matched == 0 {
		return nil, notes, errors.New("загварын толгойтой sheet олдсонгүй (\"Нэгж талбарын дугаар\" зэрэг үндсэн баганууд)")
	}
	return rows, notes, nil
}

func readReportSheet(file, sheet string, t sheetTable) ([]ReportRow, []string, bool) {
	columnCount := len(columns)
	for _, r := range t.text[:min(len(t.text), 22)] {
		columnCount = max(columnCount, len(r))
	}
	headerRow := 0
	for r := 1; r <= min(20, len(t.text)) && headerRow == 0; r++ {
		for c := 1; c <= columnCount; c++ {
			if matchesColumn("parcelID", t.cell(t.text, r, c)) {
				headerRow = r
				break
			}
		}
	}
	if headerRow == 0 {
		return nil, nil, false
	}

	// Хоёр мөр толгой: дээд нь бүлэг (нэгтгэсэн нүд — хоосон бол өмнөхийг
	// үргэлжлүүлнэ), доод нь дэд багана.
	header := map[string]int{}
	taken := map[string]bool{}
	lastGroup := ""
	for c := 1; c <= columnCount; c++ {
		g := normalize(t.cell(t.text, headerRow, c))
		if g != "" {
			lastGroup = g
		}
		group := g
		if group == "" {
			group = lastGroup
		}
		sub := normalize(t.cell(t.text, headerRow+1, c))
		for _, spec := range columns {
			if taken[spec.key] {
				continue
			}
			if (sub != "" && matchesColumn(spec.key, sub)) || (group != "" && matchesColumn(spec.key, group)) {
				header[spec.key] = c
				taken[spec.key] = true
				break
			}
		}
	}
	var missing []string
	for _, k := range requiredColumns {
		if _, ok := header[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, []string{fmt.Sprintf("%s / %s: загвар таарахгүй тул алгасав (дутуу багана: %s)",
			file, sheet, strings.Join(missing, ", "))}, false
	}

	text := func(r int, key string) string {
		if c, ok := header[key]; ok {
			return normalize(t.cell(t.text, r, c))
		}
		return ""
	}
	raw := func(r int, key string) string {
		if c, ok := header[key]; ok {
			return normalize(t.cell(t.raw, r, c))
		}
		return ""
	}

	var sheetYear *int
	if m := reYear.FindStringSubmatch(sheet); m != nil {
		if y, _ := strconv.Atoi(m[1]); y >= 1990 && y <= 2100 {
			sheetYear = &y
		}
	}

	var rows []ReportRow
	var notes []string
	blank := 0
	for r := headerRow + 2; r <= len(t.text); r++ {
		alive := text(r, "projectName") != "" || text(r, "holderName") != "" || text(r, "address") != "" ||
			parseAmount(raw(r, "totalAmount")) > 0 || parseAmount(raw(r, "landAmount")) > 0 ||
			parseAmount(raw(r, "realStateAmount")) > 0 || parseAmount(raw(r, "propertyAmount")) > 0
		if !alive {
			if blank++; blank >= emptyRowLimit {
				break
			}
			continue
		}
		blank = 0

		parcelID, oldParcelID, err := parseParcelIDs(raw(r, "parcelID"))
		if err != nil {
			notes = append(notes, fmt.Sprintf("%s / %s, %d-р мөр: %s", file, sheet, r, err.Error()))
		}
		if !reDigits6.MatchString(parcelID) {
			parcelID = ""
		}
		dbChange, extra := text(r, "dbChangeNote"), text(r, "note")
		decree := text(r, "decree")
		rows = append(rows, ReportRow{
			SourceFile: file, SourceSheet: sheet, ExcelRow: r,
			RowNo: text(r, "rowNo"), ParcelID: parcelID, OldParcelID: oldParcelID, SheetYear: sheetYear,
			ProjectName: text(r, "projectName"), ConstructionType: text(r, "constructionType"),
			DecisionNo: text(r, "decisionNo"), ListNo: text(r, "listNo"), FundingSource: text(r, "fundingSource"),
			Decree: decree, DecreeDate: parseDate(decree),
			HolderName: text(r, "holderName"), HolderRegisterNo: cleanRegisterNo(text(r, "holderRegisterNo")),
			Address:        text(r, "address"),
			AreaM2:         parseOptionalNumber(raw(r, "areaM2")),
			RightTypeLabel: text(r, "rightType"), CertificateNo: text(r, "certificateNo"),
			AffectedAreaM2:        parseOptionalNumber(raw(r, "affectedAreaM2")),
			LandAmount:            parseAmount(raw(r, "landAmount")),
			AssetNumber:           text(r, "assetNumber"),
			RealStateAmount:       parseAmount(raw(r, "realStateAmount")),
			PropertyAmount:        parseAmount(raw(r, "propertyAmount")),
			TotalAmount:           parseAmount(raw(r, "totalAmount")),
			CompensationTypeLabel: text(r, "compensationType"),
			DBChangeNote:          joinNonEmpty(" - ", dbChange, extra),
			ChangedParcelID:       parcelKey(raw(r, "changedParcelID")),
			Note:                  joinNonEmpty("; ", dbChange, extra),
		})
	}
	return rows, notes, true
}

func joinNonEmpty(sep string, parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}

// mergeReports — нэг дугаар олон файл/sheet-д давтагдвал НЭГ мөр болгоно
// (хоосон утгаар өмнөхийг дарж бичихгүй). duplicates: дугаар → эх сурвалжууд.
func mergeReports(batches [][]ReportRow) ([]ReportRow, map[string][]string) {
	byKey := map[string]*ReportRow{}
	var order []string
	sources := map[string][]string{}
	for _, batch := range batches {
		for i := range batch {
			row := batch[i]
			key := row.ParcelID
			if key == "" {
				key = fmt.Sprintf("%s|%s|%d", row.SourceFile, row.SourceSheet, row.ExcelRow)
			}
			src := row.SourceFile + " / " + row.SourceSheet
			if !containsString(sources[key], src) {
				sources[key] = append(sources[key], src)
			}
			prev, ok := byKey[key]
			if !ok {
				r := row
				byKey[key] = &r
				order = append(order, key)
				continue
			}
			merged := row
			pick := func(a, b string) string {
				if a != "" {
					return a
				}
				return b
			}
			pickNum := func(a, b float64) float64 {
				if a != 0 {
					return a
				}
				return b
			}
			pickPtr := func(a, b *float64) *float64 {
				if a != nil {
					return a
				}
				return b
			}
			merged.OldParcelID = pick(row.OldParcelID, prev.OldParcelID)
			merged.ProjectName = pick(row.ProjectName, prev.ProjectName)
			merged.ConstructionType = pick(row.ConstructionType, prev.ConstructionType)
			merged.DecisionNo = pick(row.DecisionNo, prev.DecisionNo)
			merged.ListNo = pick(row.ListNo, prev.ListNo)
			merged.FundingSource = pick(row.FundingSource, prev.FundingSource)
			merged.HolderName = pick(row.HolderName, prev.HolderName)
			merged.HolderRegisterNo = pick(row.HolderRegisterNo, prev.HolderRegisterNo)
			merged.Address = pick(row.Address, prev.Address)
			merged.RightTypeLabel = pick(row.RightTypeLabel, prev.RightTypeLabel)
			merged.CertificateNo = pick(row.CertificateNo, prev.CertificateNo)
			merged.AssetNumber = pick(row.AssetNumber, prev.AssetNumber)
			merged.Decree = pick(row.Decree, prev.Decree)
			merged.DecreeDate = pick(row.DecreeDate, prev.DecreeDate)
			merged.DBChangeNote = pick(row.DBChangeNote, prev.DBChangeNote)
			merged.ChangedParcelID = pick(row.ChangedParcelID, prev.ChangedParcelID)
			merged.CompensationTypeLabel = pick(row.CompensationTypeLabel, prev.CompensationTypeLabel)
			merged.AreaM2 = pickPtr(row.AreaM2, prev.AreaM2)
			merged.AffectedAreaM2 = pickPtr(row.AffectedAreaM2, prev.AffectedAreaM2)
			merged.LandAmount = pickNum(row.LandAmount, prev.LandAmount)
			merged.RealStateAmount = pickNum(row.RealStateAmount, prev.RealStateAmount)
			merged.PropertyAmount = pickNum(row.PropertyAmount, prev.PropertyAmount)
			merged.TotalAmount = pickNum(row.TotalAmount, prev.TotalAmount)
			merged.Note = joinNonEmpty("; ", prev.Note, row.Note)
			if merged.SheetYear == nil {
				merged.SheetYear = prev.SheetYear
			}
			byKey[key] = &merged
		}
	}
	out := make([]ReportRow, 0, len(order))
	dups := map[string][]string{}
	for _, key := range order {
		out = append(out, *byKey[key])
		if len(sources[key]) > 1 {
			dups[key] = sources[key]
		}
	}
	return out, dups
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
