package tags

import (
	"fmt"

	"github.com/flosch/pongo2/v6"
	"kandaoni.com/anqicms/provider"
)

// moduleList 内容模型列表标签。
//
//	{% moduleList modules %}
//	{% for item in modules.List %}
//	<a href="{{ item.SearchLink }}">{{ item.Title }}</a>
//	{% endfor %}
//	{% endmoduleList %}
//
// 除列表外还返回 SearchLink（全部内容的搜索地址），
// 供「搜索框」的模型下拉直接使用，避免模板里硬编码搜索路由。
type tagModuleListNode struct {
	name    string
	wrapper *pongo2.NodeWrapper
}

// moduleListItem 下拉里的一项
type moduleListItem struct {
	Id         uint
	Title      string
	Name       string
	UrlToken   string
	IsSystem   int
	SearchLink string
}

// moduleListResult 标签输出：全量搜索地址 + 模型列表
type moduleListResult struct {
	SearchLink string
	List       []*moduleListItem
}

func (node *tagModuleListNode) Execute(ctx *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {
	currentSite, _ := ctx.Public["website"].(*provider.Website)
	if currentSite == nil || currentSite.DB == nil {
		return nil
	}

	modules, err := currentSite.GetModules()
	if err != nil {
		return nil
	}

	result := &moduleListResult{
		SearchLink: currentSite.GetUrl(provider.PatternSearch, nil, 0),
		List:       make([]*moduleListItem, 0, len(modules)),
	}
	for _, item := range modules {
		if item.Status != 1 {
			continue
		}
		result.List = append(result.List, &moduleListItem{
			Id:         item.Id,
			Title:      item.Title,
			Name:       item.Name,
			UrlToken:   item.UrlToken,
			IsSystem:   item.IsSystem,
			SearchLink: currentSite.GetUrl(provider.PatternSearch, map[string]interface{}{"module": item.UrlToken}, 0),
		})
	}

	ctx.Private[node.name] = result
	if node.wrapper != nil {
		node.wrapper.Execute(ctx, writer)
	}

	return nil
}

func TagModuleListParser(doc *pongo2.Parser, _ *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	tagNode := &tagModuleListNode{}

	nameToken := arguments.MatchType(pongo2.TokenIdentifier)
	if nameToken == nil {
		return nil, arguments.Error("moduleList-tag needs a accept name.", nil)
	}
	tagNode.name = nameToken.Val

	for arguments.Remaining() > 0 {
		return nil, arguments.Error("Malformed moduleList-tag arguments.", nil)
	}

	wrapper, endtagargs, err := doc.WrapUntilTag("endmoduleList")
	if err != nil {
		return nil, err
	}
	if endtagargs.Remaining() > 0 {
		endtagnameToken := endtagargs.MatchType(pongo2.TokenIdentifier)
		if endtagnameToken != nil && endtagnameToken.Val != nameToken.Val {
			return nil, endtagargs.Error(fmt.Sprintf("Name for 'endmoduleList' must equal to 'moduleList'-tag's name ('%s' != '%s').",
				nameToken.Val, endtagnameToken.Val), nil)
		}
		if endtagnameToken == nil || endtagargs.Remaining() > 0 {
			return nil, endtagargs.Error("Either no or only one argument (identifier) allowed for 'endmoduleList'.", nil)
		}
	}
	tagNode.wrapper = wrapper

	return tagNode, nil
}
