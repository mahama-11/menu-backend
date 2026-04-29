package templatecenter

import (
	"bufio"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	templateHeadingRE = regexp.MustCompile(`^####\s+(TPL-[A-Z]+-\d+)\s+·\s+(.+)$`)
	imageRE           = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	digitsRE          = regexp.MustCompile(`\d+`)
)

func ParseTemplateLibraryMarkdown(markdown string) (*TemplateSeedLibrary, error) {
	lines := splitLines(markdown)
	library := &TemplateSeedLibrary{
		Version: "parsed-from-markdown",
		Meta: TemplateMetaResult{
			Cuisines:  parseMetaOptions(lines, "### 2.1 菜系（Cuisine）"),
			DishTypes: parseMetaOptions(lines, "### 2.2 菜品类型（Dish Type）"),
			Platforms: parsePlatformOptions(lines),
			Moods:     parseMetaOptions(lines, "### 2.4 视觉风格（Mood）"),
			Plans: []TemplateMetaOption{
				{ID: "basic", Label: "Basic"},
				{ID: "pro", Label: "Pro"},
				{ID: "growth", Label: "Growth"},
			},
		},
	}
	templates, err := parseTemplateSections(lines, library.Meta.Platforms)
	if err != nil {
		return nil, err
	}
	library.Templates = templates
	return library, nil
}

func splitLines(markdown string) []string {
	scanner := bufio.NewScanner(strings.NewReader(markdown))
	lines := make([]string, 0, 256)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func parseMetaOptions(lines []string, sectionTitle string) []TemplateMetaOption {
	start := findLine(lines, sectionTitle)
	if start < 0 {
		return nil
	}
	tableLines := collectTableLines(lines[start+1:])
	rows := parseMarkdownTable(tableLines)
	options := make([]TemplateMetaOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, TemplateMetaOption{
			ID:          firstNonEmpty(row["ID"], row["Id"]),
			Label:       firstNonEmpty(row["英文"], row["平台名称"], row["中文"], row["泰语"]),
			Description: firstNonEmpty(row["适用场景"], row["典型菜品"]),
		})
	}
	return options
}

func parsePlatformOptions(lines []string) []TemplatePlatformOption {
	start := findLine(lines, "### 2.3 推广平台（Platform）")
	if start < 0 {
		return nil
	}
	rows := parseMarkdownTable(collectTableLines(lines[start+1:]))
	options := make([]TemplatePlatformOption, 0, len(rows))
	for _, row := range rows {
		width, height := parseSize(row["尺寸"])
		options = append(options, TemplatePlatformOption{
			ID:     row["ID"],
			Label:  row["平台名称"],
			Width:  width,
			Height: height,
			Ratio:  row["比例"],
			Format: strings.ToLower(row["格式"]),
		})
	}
	return options
}

func parseTemplateSections(lines []string, platforms []TemplatePlatformOption) ([]TemplateSeed, error) {
	platformByLabel := map[string]string{}
	for _, item := range platforms {
		for _, alias := range platformAliases(item) {
			platformByLabel[alias] = item.ID
		}
	}
	cuisineBySection := map[string]string{
		"泰国菜系": "thai",
		"日本料理": "japanese",
		"中国菜系": "chinese",
		"西餐":   "western",
		"韩国料理": "korean",
		"海鲜":   "seafood",
		"饮品 / 甜品": "dessert",
	}
	currentCuisine := ""
	var items []TemplateSeed
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "### ") {
			for key, value := range cuisineBySection {
				if strings.Contains(line, key) {
					currentCuisine = value
					break
				}
			}
		}
		matches := templateHeadingRE.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		templateID := matches[1]
		title := strings.TrimSpace(matches[2])
		j := i + 1
		sectionLines := make([]string, 0, 32)
		for ; j < len(lines); j++ {
			next := strings.TrimSpace(lines[j])
			if strings.HasPrefix(next, "#### ") || strings.HasPrefix(next, "### ") {
				break
			}
			sectionLines = append(sectionLines, lines[j])
		}
		item := parseSingleTemplateSection(templateID, title, currentCuisine, sectionLines, platformByLabel)
		items = append(items, item)
		i = j - 1
	}
	return items, nil
}

func parseSingleTemplateSection(templateID, title, cuisine string, lines []string, platformByLabel map[string]string) TemplateSeed {
	row := parseDetailTable(collectTableLines(lines))
	item := TemplateSeed{
		ID:          templateID,
		Slug:        buildTemplateSlug(templateID),
		Name:        buildTemplateName(title, row),
		Description: buildTemplateDescription(title, row),
		Cuisine:     mapCuisine(templateID, cuisine),
		DishType:    inferDishType(templateID, title, row),
		Plan:        parsePlan(firstNonEmpty(row["套餐权限"], "Basic")),
		CreditsCost: parseCredits(firstNonEmpty(row["积分消耗"], "10")),
		Platforms:   parsePlatforms(row["适配平台"], platformByLabel),
		Moods:       parseMoods(row["视觉风格"]),
		Tags:        buildTemplateTags(templateID, title, row),
		Layout:      row["布局"],
		Lighting:    row["光效"],
		Props:       parseDelimitedList(row["道具"], "、", "，", ","),
		PromptTemplates: map[string]string{
			"en": parsePrompt(lines, "EN"),
		},
		CopyTemplates: map[string]any{},
		Hashtags:      map[string][]string{},
		ExportSpecs:   map[string]any{},
		InputSchema:   map[string]any{},
		ExecutionProfile: map[string]any{},
		Examples:      parseExampleImages(lines),
		Metadata: map[string]any{
			"source": "template_library_markdown",
		},
	}
	if len(item.Platforms) == 0 {
		item.Platforms = inferPlatformsFromTitle(title, platformByLabel)
	}
	return item
}

func parseDetailTable(lines []string) map[string]string {
	rows := parseMarkdownTable(lines)
	if len(rows) == 0 {
		return map[string]string{}
	}
	details := make(map[string]string, len(rows))
	for _, row := range rows {
		key := firstNonEmpty(row["项目"], row["Item"], row["字段"], row["Field"])
		value := firstNonEmpty(row["内容"], row["Value"], row["值"])
		if key == "" {
			continue
		}
		details[key] = value
	}
	if len(details) > 0 {
		return details
	}
	return rows[0]
}

func parseMarkdownTable(lines []string) []map[string]string {
	if len(lines) < 2 {
		return nil
	}
	headers := parseTableRow(lines[0])
	if len(headers) == 0 {
		return nil
	}
	rows := make([]map[string]string, 0, len(lines)-2)
	for _, line := range lines[2:] {
		values := parseTableRow(line)
		if len(values) == 0 {
			continue
		}
		row := map[string]string{}
		for idx, header := range headers {
			if idx < len(values) {
				row[header] = values[idx]
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		return nil
	}
	parts := strings.Split(line, "|")
	values := make([]string, 0, len(parts))
	for _, part := range parts[1 : len(parts)-1] {
		cell := strings.TrimSpace(strings.Trim(part, "* "))
		cell = strings.Trim(cell, "`")
		cell = strings.TrimSpace(strings.ReplaceAll(cell, "\u00a0", " "))
		values = append(values, cell)
	}
	return values
}

func collectTableLines(lines []string) []string {
	out := make([]string, 0, 8)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(out) > 0 && !strings.HasPrefix(trimmed, "|") {
			break
		}
		if strings.HasPrefix(trimmed, "|") {
			out = append(out, trimmed)
		}
	}
	return out
}

func findLine(lines []string, target string) int {
	for idx, line := range lines {
		if strings.TrimSpace(line) == target {
			return idx
		}
	}
	return -1
}

func parseSize(value string) (int, int) {
	parts := strings.Split(strings.TrimSpace(value), "×")
	if len(parts) != 2 {
		return 0, 0
	}
	width, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return width, height
}

func parsePlatforms(raw string, platformByLabel map[string]string) []string {
	parts := parseDelimitedList(raw, "·", "•")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		label := normalizeTableValue(part)
		if id, ok := platformByLabel[label]; ok {
			items = append(items, id)
			continue
		}
		if id, ok := platformByLabel[compactPlatformAlias(label)]; ok {
			items = append(items, id)
		}
	}
	return uniqueStrings(items)
}

func inferPlatformsFromTitle(title string, platformByLabel map[string]string) []string {
	items := make([]string, 0, 2)
	for label, id := range platformByLabel {
		if strings.Contains(title, label) {
			items = append(items, id)
		}
	}
	return uniqueStrings(items)
}

func parseMoods(raw string) []string {
	parts := parseDelimitedList(raw, "+", "＋", "·")
	labelToID := map[string]string{
		"食欲感": "appetizing",
		"精致感": "elegant",
		"节日感": "festive",
		"日常感": "casual",
		"高端感": "premium",
		"健康感": "healthy",
		"辣感":  "spicy",
		"清新感": "fresh",
	}
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if id, ok := labelToID[normalizeTableValue(part)]; ok {
			items = append(items, id)
		}
	}
	return uniqueStrings(items)
}

func parsePrompt(lines []string, lang string) string {
	header := fmt.Sprintf("**AI 图像提示词（%s）：**", lang)
	for idx, line := range lines {
		if strings.TrimSpace(line) != header {
			continue
		}
		for j := idx + 1; j < len(lines); j++ {
			next := strings.TrimSpace(lines[j])
			if prompt, ok := strings.CutPrefix(next, ">"); ok {
				return strings.TrimSpace(prompt)
			}
			if next != "" {
				break
			}
		}
	}
	return ""
}

func parseExampleImages(lines []string) []TemplateExampleSeed {
	items := make([]TemplateExampleSeed, 0)
	for _, line := range lines {
		matches := imageRE.FindAllStringSubmatch(line, -1)
		for idx, match := range matches {
			items = append(items, TemplateExampleSeed{
				ExampleType: "preview",
				Title:       match[1],
				PreviewURL:  match[2],
				SortOrder:   idx + 1,
			})
		}
	}
	return items
}

func buildTemplateSlug(templateID string) string {
	base := strings.ToLower(templateID)
	replacer := strings.NewReplacer("×", "-", "·", "-", "/", "-", " ", "-", "--", "-", "(", "", ")", "", ":", "", "'", "", "_", "-", "#", "")
	base = replacer.Replace(base)
	base = regexp.MustCompile(`[^a-z0-9\-]+`).ReplaceAllString(base, "")
	base = strings.Trim(base, "-")
	if base == "" {
		base = strings.ToLower(strings.ReplaceAll(templateID, "_", "-"))
	}
	return base
}

func buildTemplateName(title string, row map[string]string) string {
	if dish := row["菜品"]; dish != "" {
		parts := strings.Split(dish, "/")
		if len(parts) >= 2 {
			return strings.TrimSpace(parts[1]) + " × " + title
		}
	}
	return title
}

func buildTemplateDescription(title string, row map[string]string) string {
	return firstNonEmpty(row["用途"], row["适配平台"], title)
}

func buildTemplateTags(templateID, title string, row map[string]string) []string {
	tags := []string{strings.ToLower(strings.ReplaceAll(templateID, "-", "_"))}
	for _, item := range parseMoods(row["视觉风格"]) {
		tags = append(tags, item)
	}
	for _, platform := range parseDelimitedList(row["适配平台"], "·") {
		tags = append(tags, slugWord(platform))
	}
	if title != "" {
		tags = append(tags, slugWord(title))
	}
	return uniqueStrings(tags)
}

func mapCuisine(templateID, fallback string) string {
	if fallback != "" {
		return fallback
	}
	switch {
	case strings.HasPrefix(templateID, "TPL-TH-"):
		return "thai"
	case strings.HasPrefix(templateID, "TPL-JP-"):
		return "japanese"
	case strings.HasPrefix(templateID, "TPL-CN-"):
		return "chinese"
	case strings.HasPrefix(templateID, "TPL-WS-"):
		return "western"
	case strings.HasPrefix(templateID, "TPL-KR-"):
		return "korean"
	case strings.HasPrefix(templateID, "TPL-SF-"):
		return "seafood"
	case strings.HasPrefix(templateID, "TPL-DS-"):
		return "dessert"
	default:
		return "western"
	}
}

func inferDishType(templateID, title string, row map[string]string) string {
	searchText := strings.Join([]string{title, row["菜品"], row["用途"]}, " ")
	matches := []struct {
		pattern string
		value   string
	}{
		{"冬阴功", "soup"},
		{"Tom Yum", "soup"},
		{"寿司", "signature"},
		{"Sushi", "signature"},
		{"糯米饭", "dessert"},
		{"Sticky Rice", "dessert"},
		{"奶茶", "drink"},
		{"Tea", "drink"},
		{"蛋糕", "dessert"},
		{"Cake", "dessert"},
		{"沙拉", "salad"},
		{"Salad", "salad"},
		{"沙爹", "grill"},
		{"BBQ", "grill"},
		{"拉面", "rice_noodle"},
		{"Ramen", "rice_noodle"},
		{"套餐", "set_meal"},
		{"Platter", "set_meal"},
		{"菜单", "set_meal"},
		{"Menu", "set_meal"},
	}
	for _, item := range matches {
		if strings.Contains(searchText, item.pattern) {
			return item.value
		}
	}
	switch {
	case strings.HasPrefix(templateID, "TPL-DS-001"):
		return "drink"
	case strings.HasPrefix(templateID, "TPL-TH-005"):
		return "salad"
	case strings.HasPrefix(templateID, "TPL-TH-004"), strings.HasPrefix(templateID, "TPL-KR-001"):
		return "grill"
	case strings.HasPrefix(templateID, "TPL-JP-002"):
		return "rice_noodle"
	case strings.HasPrefix(templateID, "TPL-DS-002"), strings.HasPrefix(templateID, "TPL-TH-003"):
		return "dessert"
	case strings.HasPrefix(templateID, "TPL-WS-001"), strings.HasPrefix(templateID, "TPL-SF-001"), strings.HasPrefix(templateID, "TPL-MENU-001"):
		return "set_meal"
	default:
		return "signature"
	}
}

func parsePlan(raw string) string {
	value := strings.ToLower(raw)
	switch {
	case strings.Contains(value, "growth"):
		return "growth"
	case strings.Contains(value, "pro"):
		return "pro"
	default:
		return "basic"
	}
}

func parseCredits(raw string) int64 {
	match := digitsRE.FindString(raw)
	if match == "" {
		return 10
	}
	value, _ := strconv.ParseInt(match, 10, 64)
	if value == 0 {
		return 10
	}
	return value
}

func parseDelimitedList(raw string, separators ...string) []string {
	value := normalizeTableValue(raw)
	if value == "" {
		return nil
	}
	replacerArgs := make([]string, 0, len(separators)*2)
	for _, sep := range separators {
		replacerArgs = append(replacerArgs, sep, "|")
	}
	value = strings.NewReplacer(replacerArgs...).Replace(value)
	parts := strings.Split(value, "|")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeTableValue(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

func slugWord(value string) string {
	value = strings.ToLower(normalizeTableValue(value))
	replacer := strings.NewReplacer(" ", "_", "×", "_", "·", "_", "/", "_", "-", "_")
	return strings.Trim(replacer.Replace(value), "_")
}

func normalizeTableValue(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "* "))
	value = strings.Trim(value, "`")
	value = strings.ReplaceAll(value, "\u00a0", " ")
	return strings.TrimSpace(value)
}

func platformAliases(item TemplatePlatformOption) []string {
	aliases := []string{normalizeTableValue(item.Label)}
	switch item.ID {
	case "grabfood":
		aliases = append(aliases, "GrabFood", "GrabFood Banner")
	case "foodpanda":
		aliases = append(aliases, "foodpanda", "foodpanda Banner")
	case "qr_menu":
		aliases = append(aliases, "QR Menu", "QR Code Menu")
	case "print_a4":
		aliases = append(aliases, "Print A4", "Print Flyer A4")
	case "print_a5":
		aliases = append(aliases, "Print A5", "Print Flyer A5")
	}
	out := make([]string, 0, len(aliases)*2)
	for _, alias := range aliases {
		alias = normalizeTableValue(alias)
		if alias == "" {
			continue
		}
		out = append(out, alias, compactPlatformAlias(alias))
	}
	return uniqueStrings(out)
}

func compactPlatformAlias(value string) string {
	value = strings.ToLower(normalizeTableValue(value))
	value = strings.NewReplacer(" banner", "", " flyer", "", " code", "").Replace(value)
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
