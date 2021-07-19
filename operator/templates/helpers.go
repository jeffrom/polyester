package templates

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

func tmplHelpers() template.FuncMap {
	fns := template.FuncMap{
		// "sopsDecrypt": sopsDecrypt,
		"string": toString,
	}

	spfns := sprig.HermeticTxtFuncMap()
	for k, fn := range spfns {
		fns[k] = fn
	}

	return fns
}

// func sopsDecrypt(

func toString(i interface{}) string {
	switch v := i.(type) {
	case string:
		return v
	case []uint8:
		var b strings.Builder
		for _, ch := range v {
			b.WriteRune(rune(ch))
		}
		return b.String()
	default:
		panic(fmt.Sprintf("templates: string conversion for %T not supported", i))
	}
}
