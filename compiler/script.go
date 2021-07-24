package compiler

import (
	"bytes"
	"html/template"

	"github.com/jeffrom/polyester/compiler/shell"
)

var planBoilerplate = template.Must(template.New("boilerplate").Parse(`
# --- START polyester script boilerplate
{{- with $file := .SelfFile }}
alias polyester={{ $file }}
{{- end }}
alias P=polyester

# --- END polyester script boilerplate
`))

// annotatePlanScript adds boilerplate to plan script before executing them.
// All it currently adds is: alias P polyester.
// If selfFile is not polyester (ie in a test), an alias polyester=selfFile
// will be added.
func annotatePlanScript(planb []byte, selfFile string) ([]byte, error) {
	psh, err := shell.Parse(bytes.NewReader(planb))
	if err != nil {
		return nil, err
	}
	planb, err = psh.ConvertShellOp()
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	err = planBoilerplate.Execute(buf, struct {
		SelfFile string
	}{
		SelfFile: selfFile,
	})
	if err != nil {
		return nil, err
	}
	planDeclBoilerplate := buf.Bytes()
	res := make([]byte, len(planb)+len(planDeclBoilerplate))
	// if the first line is a shebang, put the boilerplate on the second line
	if bytes.HasPrefix(planb, []byte("#!")) {
		idx := bytes.Index(planb, []byte("\n"))
		// this script is just one-line shebang without a newline
		if idx == -1 || len(planb) < idx+1 {
			return planb, nil
		}
		firstLine := planb[:idx+1]
		copy(res, firstLine)
		copy(res[idx+1:], planDeclBoilerplate)
		copy(res[idx+1+len(planDeclBoilerplate):], planb[idx+1:])
		return res, nil
	}

	copy(res, planDeclBoilerplate)
	copy(res[len(planDeclBoilerplate):], planb)
	return res, nil
}
