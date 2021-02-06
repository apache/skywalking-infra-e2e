package verifier

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/apache/skywalking-infra-e2e/third-party/template"
)

// funcMap produces the custom function map.
// Use this to pass the functions into the template engine:
// 	tpl := template.New("foo").Funcs(funcMap()))
func funcMap() template.FuncMap {
	fm := make(map[string]interface{}, len(customFuncMap))
	for k, v := range customFuncMap {
		fm[k] = v
	}
	return template.FuncMap(fm)
}

var customFuncMap = map[string]interface{}{
	// Basic:
	"notEmpty": notEmpty,

	// Encoding:
	"b64enc": base64encode,

	// Regex:
	"regexp": regexpMatch,
}

func base64encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func notEmpty(s string) string {
	if len(strings.TrimSpace(s)) > 0 {
		return s
	}
	return fmt.Sprintf(`<"%s" is empty, wanted is not empty>`, s)
}

func regexpMatch(s, pattern string) string {
	matched, err := regexp.MatchString(pattern, s)
	if err != nil {
		return fmt.Sprintf(`<"%s">`, err)
	}
	if !matched {
		return fmt.Sprintf(`<"%s" does not match the pattern %s">`, s, pattern)
	}
	return s
}
