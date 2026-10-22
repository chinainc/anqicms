package tags

import (
	"github.com/flosch/pongo2/v6"
	"kandaoni.com/anqicms/provider"
)

// diyPage 输出可视化装修的页面区块。
//
//	{% diyPage home %}
//
// 没有配置时输出空串，模板可用 {% if %} 回退到默认内容。
type tagDiyPageNode struct {
	key string
}

func (node *tagDiyPageNode) Execute(ctx *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {
	currentSite, _ := ctx.Public["website"].(*provider.Website)
	if currentSite == nil || currentSite.DB == nil {
		return nil
	}
	writer.WriteString(currentSite.RenderDiyPage(node.key))
	return nil
}

func TagDiyPageParser(doc *pongo2.Parser, _ *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	tagNode := &tagDiyPageNode{}
	// 支持 {% diyPage home %} 与 {% diyPage "home" %}
	if tok := arguments.MatchType(pongo2.TokenIdentifier); tok != nil {
		tagNode.key = tok.Val
	} else if tok := arguments.MatchType(pongo2.TokenString); tok != nil {
		tagNode.key = tok.Val
	} else {
		return nil, arguments.Error("diyPage-tag needs a page key.", nil)
	}
	if arguments.Remaining() > 0 {
		return nil, arguments.Error("Malformed diyPage-tag arguments.", nil)
	}
	return tagNode, nil
}
