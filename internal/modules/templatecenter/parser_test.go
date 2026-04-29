package templatecenter

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseTemplateLibraryMarkdown_DetailFields(t *testing.T) {
	library := parseTemplateLibraryDocForTest(t)

	tomYum := findTemplateSeedForTest(t, library, "TPL-TH-001")
	if got, want := tomYum.Platforms, []string{"instagram_feed", "line_oa"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected platforms: %#v", got)
	}
	if got, want := tomYum.Moods, []string{"appetizing", "spicy"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected moods: %#v", got)
	}
	if tomYum.Layout == "" || tomYum.Lighting == "" || len(tomYum.Props) != 3 {
		t.Fatalf("expected parsed design fields: %+v", tomYum)
	}
	if tomYum.PromptTemplates["en"] == "" {
		t.Fatalf("expected english prompt: %+v", tomYum.PromptTemplates)
	}
}

func TestParseTemplateLibraryMarkdown_PlatformAliases(t *testing.T) {
	library := parseTemplateLibraryDocForTest(t)

	menuTemplate := findTemplateSeedForTest(t, library, "TPL-MENU-001")
	if got, want := menuTemplate.Platforms, []string{"print_a4", "qr_menu"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected menu platforms: %#v", got)
	}

	ramen := findTemplateSeedForTest(t, library, "TPL-JP-002")
	expected := map[string]bool{"facebook_post": true, "grabfood": true, "instagram_feed": true}
	if len(ramen.Platforms) != len(expected) {
		t.Fatalf("unexpected ramen platforms: %#v", ramen.Platforms)
	}
	for _, platform := range ramen.Platforms {
		if !expected[platform] {
			t.Fatalf("unexpected ramen platform %q in %#v", platform, ramen.Platforms)
		}
	}
}

func TestParseTemplateLibraryMarkdown_ExampleImages(t *testing.T) {
	markdown := `
### 2.1 菜系（Cuisine）
| ID | 泰语 | 中文 | 英文 |
|----|------|------|------|
| thai | 泰 | 泰国菜 | Thai Cuisine |

### 2.2 菜品类型（Dish Type）
| ID | 中文 | 典型菜品 |
|----|------|---------|
| signature | 招牌主菜 | 绿咖喱 |

### 2.3 推广平台（Platform）
| ID | 平台名称 | 尺寸 | 比例 | 格式 |
|----|----------|------|------|------|
| instagram_feed | Instagram Feed | 1080×1080 | 1:1 | JPG |

### 2.4 视觉风格（Mood）
| ID | 中文 | 适用场景 |
|----|------|---------|
| appetizing | 食欲感 | 日常推广 |

### 🇹🇭 泰国菜系
#### TPL-TH-999 · 测试模板
| 项目 | 内容 |
|------|------|
| 菜品 | 测试 / 测试菜 / Test Dish |
| 适配平台 | Instagram Feed |
| 视觉风格 | 食欲感 |
| 布局 | 中心构图 |
| 光效 | 暖色光 |
| 道具 | 薄荷叶、刀叉 |
| 套餐权限 | Basic |
| 积分消耗 | 10 cr |

![封面图](https://example.com/cover.jpg)
![输出图](https://example.com/output.jpg)
`
	library, err := ParseTemplateLibraryMarkdown(markdown)
	if err != nil {
		t.Fatalf("ParseTemplateLibraryMarkdown: %v", err)
	}
	item := findTemplateSeedForTest(t, library, "TPL-TH-999")
	if len(item.Examples) != 2 {
		t.Fatalf("expected 2 examples, got %#v", item.Examples)
	}
	if item.Examples[0].PreviewURL != "https://example.com/cover.jpg" || item.Examples[1].PreviewURL != "https://example.com/output.jpg" {
		t.Fatalf("unexpected example urls: %#v", item.Examples)
	}
}

func parseTemplateLibraryDocForTest(t *testing.T) *TemplateSeedLibrary {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	docPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", "docs", "TEMPLATE_LIBRARY_DOC.md")
	payload, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read template library doc: %v", err)
	}
	library, err := ParseTemplateLibraryMarkdown(string(payload))
	if err != nil {
		t.Fatalf("ParseTemplateLibraryMarkdown: %v", err)
	}
	return library
}

func findTemplateSeedForTest(t *testing.T, library *TemplateSeedLibrary, templateID string) TemplateSeed {
	t.Helper()
	for _, item := range library.Templates {
		if item.ID == templateID {
			return item
		}
	}
	t.Fatalf("template %s not found", templateID)
	return TemplateSeed{}
}
