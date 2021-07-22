package compiler

import "github.com/jeffrom/polyester/manifest"

type intermediatePlan struct {
	*manifest.Manifest
	compiled map[string][]byte
}

func newIntermediatePlan(m *manifest.Manifest) *intermediatePlan {
	return &intermediatePlan{
		Manifest: m,
		compiled: make(map[string][]byte),
	}
}
