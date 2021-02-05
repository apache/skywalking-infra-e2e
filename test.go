package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/apache/skywalking-infra-e2e/third-party/template"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
	"log"
	"strings"
)

var funcMap = template.FuncMap{
	"b64enc": func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	},
	"notEmpty": func(s string) string {
		if len(strings.TrimSpace(s)) > 0 {
			return s
		} else {
			return fmt.Sprintf(`<"%s" is empty, wanted is not empty>`, s)
		}
	},
}

func main() {
	test(`
nodes:
  - id: VXNlcg==.0
    name: User
    type: USER
    isReal: false
  - id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    name: Your_ApplicationName
    type: Tomcat
    isReal: true
  - id: bG9jYWxob3N0Oi0x.0
    name: localhost:-1
    type: H2
    isReal: false
calls:
  - id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1-bG9jYWxob3N0Oi0x.0
    source: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    detectPoints:
      - CLIENT
    target: bG9jYWxob3N0Oi0x.0
  - id: VXNlcg==.0-WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    source: VXNlcg==.0
    detectPoints:
      - SERVER
    target: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
`, `
nodes:
  - id: {{ b64enc "User" }}.0
    name: User
    type: USER
    isReal: false
  - id: {{ b64enc "Your_ApplicationName" }}.1
    name: Your_ApplicationName
    type: Tomcat
    isReal: true
  - id: {{ $h2ID := (index .nodes 2).id }}{{ notEmpty $h2ID }}
    name: localhost:-1
    type: H2
    isReal: false
calls:
  - id: {{ notEmpty (index .calls 0).id }}
    source: {{ b64enc "Your_ApplicationName" }}.1
    target: {{ $h2ID }}
    detectPoints:
      - CLIENT
  - id: {{ b64enc "User" }}.0-{{ b64enc "Your_ApplicationName" }}.1
    source: {{ b64enc "User" }}.0
    target: {{ b64enc "Your_ApplicationName" }}.1
    detectPoints:
      - SERVER
`)

	test(`
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMg==.1
    value: 0
`, `
metrics:
{{- atLeastOnce .metrics }}
  name: {{ notEmpty .name }}
  id: {{ notEmpty .id }}
  value: {{ gt .value 0 }}
{{- end }}
`)
	test(`
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 0
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMg==.1
    value: 0
`, `
metrics:
{{- atLeastOnce .metrics }}
  name: {{ notEmpty .name }}
  id: {{ notEmpty .id }}
  value: {{ gt .value 0 }}
{{- end }}
`)
}

func test(actualData string, expectedTemplate string) {
	var actual interface{}
	yaml.Unmarshal([]byte(actualData), &actual)

	tmpl, err := template.New("test").Funcs(funcMap).Parse(expectedTemplate)
	if err != nil {
		log.Fatalf("parsing: %s", err)
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, actual)
	if err != nil {
		log.Fatalf("execution: %s", err)
	}

	var expected interface{}
	yaml.Unmarshal(b.Bytes(), &expected)

	if !cmp.Equal(expected, actual) {
		diff := cmp.Diff(expected, actual)
		println(diff)
	}
}
