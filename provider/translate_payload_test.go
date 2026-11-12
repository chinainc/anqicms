package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestTranslationPayloadOmitsEmptyGlossary 护栏：
// glossary_id 为空时不能出现在请求体里（DeepL 对空串会判为无效 glossary_id 而整单失败），
// 非空时必须原样带上。反向对照：去掉 omitempty 后第一条断言必然变红。
func TestTranslationPayloadOmitsEmptyGlossary(t *testing.T) {
	empty, err := json.Marshal(TranslationPayload{
		Text:       []string{"hello"},
		TargetLang: "EN",
		SourceLang: "ZH",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(empty), "glossary_id") {
		t.Fatalf("空术语表不应出现在请求体中，实际: %s", string(empty))
	}

	withGlossary, err := json.Marshal(TranslationPayload{
		Text:       []string{"hello"},
		TargetLang: "EN",
		SourceLang: "ZH",
		GlossaryID: "glossary-abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(withGlossary), `"glossary_id":"glossary-abc"`) {
		t.Fatalf("非空术语表必须带上，实际: %s", string(withGlossary))
	}
}
