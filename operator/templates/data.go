package templates

import (
	"os"

	"github.com/ghodss/yaml"

	"github.com/jeffrom/polyester/operator/facts"
)

// Data is the template data struct used in all template operators.
type Data struct {
	// Facts are information about the local system.
	Facts facts.Facts

	// Data is any data provided via vars/default.yaml or --data.
	Data map[string]interface{}

	// Dest is the path to the current destination file.
	Dest string

	// DestIdx is the index of the current destination file, according to the
	// argument order.
	DestIdx int
}

func (t *Templates) MergeData(dataPaths []string) (map[string]interface{}, error) {
	var datas []map[string]interface{}
	for _, p := range dataPaths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		m := make(map[string]interface{})
		if err := yaml.Unmarshal(b, &m); err != nil {
			return nil, err
		}

		datas = append(datas, m)
	}

	res := make(map[string]interface{})
	for _, data := range datas {
		for k, v := range data {
			res[k] = v
		}
	}
	return res, nil
}
