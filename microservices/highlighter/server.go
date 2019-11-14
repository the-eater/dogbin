package main

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/src-d/enry/v2"
	"github.com/valyala/fasthttp"
	"github.com/wI2L/jettison"
)

var style = styles.Register(chroma.MustNewStyle("dogbin", chroma.StyleEntries {
	chroma.Comment:                  "#7988b5",
	chroma.CommentHashbang:          "#7988b5",
	chroma.CommentMultiline:         "#7988b5",
	chroma.CommentPreproc:           "#97dbbe",
	chroma.CommentSingle:            "#7988b5",
	chroma.CommentSpecial:           "#7988b5",
	chroma.Generic:                  "#f8f2f7",
	chroma.GenericDeleted:           "#963f3f",
	chroma.GenericEmph:              "#f8f2f7 underline",
	chroma.GenericError:             "#f8f2f7",
	chroma.GenericHeading:           "#f8f2f7 bold",
	chroma.GenericInserted:          "#aee8b7 bold",
	chroma.GenericOutput:            "#44475a",
	chroma.GenericPrompt:            "#e0b8da",
	chroma.GenericStrong:            "#f8f2f7 bold",
	chroma.GenericSubheading:        "#f8f2f7 bold",
	chroma.GenericTraceback:         "#f0cee0",
	chroma.GenericUnderline:         "underline",
	chroma.Error:                    "#f07199",
	chroma.Keyword:                  "#97dbbe",
	chroma.KeywordConstant:          "#97dbbe bold",
	chroma.KeywordDeclaration:       "#bc89c9 italic",
	chroma.KeywordNamespace:         "#97dbbe",
	chroma.KeywordPseudo:            "#97dbbe",
	chroma.KeywordReserved:          "#97dbbe",
	chroma.KeywordType:              "#bc89c9",
	chroma.Literal:                  "#f8f2f7",
	chroma.LiteralDate:              "#f8f2f7",
	chroma.Name:                     "#f8f2f7",
	chroma.NameAttribute:            "#94d1a4",
	chroma.NameBuiltin:              "#bc89c9 italic",
	chroma.NameBuiltinPseudo:        "#f8f2f7",
	chroma.NameClass:                "#d1b0cc",
	chroma.NameConstant:             "#d1b0cc bold",
	chroma.NameDecorator:            "#d1b0cc",
	chroma.NameEntity:               "#d1b0cc",
	chroma.NameException:            "#d1b0cc",
	chroma.NameFunction:             "#f5da98",
	chroma.NameLabel:                "#bc89c9 italic",
	chroma.NameNamespace:            "#f8f2f7",
	chroma.NameOther:                "#f8f2f7",
	chroma.NameTag:                  "#97dbbe",
	chroma.NameVariable:             "#bc89c9 italic",
	chroma.NameVariableClass:        "#bc89c9 italic",
	chroma.NameVariableGlobal:       "#bc89c9 italic",
	chroma.NameVariableInstance:     "#bc89c9 italic",
	chroma.LiteralNumber:            "#bd93f9",
	chroma.LiteralNumberBin:         "#bd93f9",
	chroma.LiteralNumberFloat:       "#bd93f9",
	chroma.LiteralNumberHex:         "#bd93f9",
	chroma.LiteralNumberInteger:     "#bd93f9",
	chroma.LiteralNumberIntegerLong: "#bd93f9",
	chroma.LiteralNumberOct:         "#bd93f9",
	chroma.Operator:                 "#97dbbe",
	chroma.OperatorWord:             "#97dbbe",
	chroma.Other:                    "#f8f2f7",
	chroma.Punctuation:              "#f8f2f7",
	chroma.LiteralString:            "#f0a8c7",
	chroma.LiteralStringBacktick:    "#f0a8c7",
	chroma.LiteralStringChar:        "#d496b1",
	chroma.LiteralStringDoc:         "#f0a8c7",
	chroma.LiteralStringDouble:      "#f0a8c7",
	chroma.LiteralStringEscape:      "#f0a8c7",
	chroma.LiteralStringHeredoc:     "#f0a8c7",
	chroma.LiteralStringInterpol:    "#f0a8c7",
	chroma.LiteralStringOther:       "#f0a8c7",
	chroma.LiteralStringRegex:       "#f0a8c7",
	chroma.LiteralStringSingle:      "#f0a8c7",
	chroma.LiteralStringSymbol:      "#f0a8c7",
	chroma.Text:                     "#f8f2f7",
	chroma.TextWhitespace:           "#f8f2f7",
	chroma.Background:               " bg:#282a36",
}))

type HighlightResponse struct {
	Lang string `json:"lang"`
	Code string `json:"code"`
}

// TODO: move line number stuff here from dogbin/app
var formatter = html.New(html.WithClasses(), html.TabWidth(2))

var supportedLangs []string

func highlightHandler(ctx *fasthttp.RequestCtx) {
	lang := string(ctx.Request.PostArgs().Peek("lang"))
	filename := string(ctx.Request.PostArgs().Peek("filename"))
	code := string(ctx.Request.PostArgs().Peek("code"))

	var lexer chroma.Lexer
	if lang != "" {
		lexer = lexers.Get(lang)
	}
	if lexer == nil && filename != "" {
		lexer = lexers.Get(enry.GetLanguage(filename, []byte(code)))
	}
	if lexer == nil && filename != "" {
		lexer = lexers.Match(filename)
	}
	var enryLang = ""
	if lexer == nil {
		//var safe = false
		// This can be quite inaccurate but appears to still be better than chroma at some languages, e.g. Kotlin
		enryLang, _ = enry.GetLanguageByClassifier([]byte(code), supportedLangs)
		//if safe {
			lexer = lexers.Get(enryLang)
		//}
	}
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	//if lexer == nil {
	//	lexer = lexers.Get(enryLang)
	//}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	tokens, err := lexer.Tokenise(nil, code)
	if err != nil {
		ctx.Error(fmt.Sprint(err), fasthttp.StatusInternalServerError)
		return
	}
	buf := bytes.Buffer{}
	err = formatter.Format(&buf, style, tokens)
	if err != nil {
		ctx.Error(fmt.Sprint(err), fasthttp.StatusInternalServerError)
		return
	}
	json, err := jettison.Marshal(HighlightResponse{
		Lang: lexer.Config().Name,
		Code: buf.String(),
	})
	if err != nil {
		ctx.Error(fmt.Sprint(err), fasthttp.StatusInternalServerError)
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetBody(json)
}

func main() {
	for _, l := range lexers.Registry.Lexers {
		config := l.Config()
		lang, ok := enry.GetLanguageByAlias(config.Name)
		if ok {
			supportedLangs = append(supportedLangs, lang)
		} else {
			for _, a := range config.Aliases {
				lang, ok := enry.GetLanguageByAlias(a)
				if ok {
					supportedLangs = append(supportedLangs, lang)
				}
			}
		}
	}
	if err := fasthttp.ListenAndServe(":8080", highlightHandler); err != nil {
		fmt.Println(err)
	}
}
