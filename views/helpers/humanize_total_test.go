package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTotal(t *testing.T) {
	t.Parallel()

	cas := map[string]string{
		"47.08":    "47.08",
		"999":      "999",
		"99.5":     "99.5",
		"999.5":    "999",
		"1337.65":  "1\u202f337",
		"93347.28": "93\u202f347",
		"611125":   "611\u202f125",
		"1234567":  "1\u202f234\u202f567",
		"-12345.6": "-12\u202f345",
		"N/A":      "N/A",
		"4:37":     "4:37",
		"3d 17h":   "3d 17h",
		"":         "",
	}

	for entree, attendu := range cas {
		assert.Equal(t, attendu, Total(entree), "Total(%q)", entree)
	}
}
