package verifier

import "testing"

func TestVerify(t *testing.T) {
	type args struct {
		actualData       string
		expectedTemplate string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "should contain two elements",
			args: args{
				actualData: `
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    value: 2
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 0 }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 1 }}
{{- end }}
`,
			},
			wantErr: false,
		},
		{
			name: "fail to contain two elements",
			args: args{
				actualData: `
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    value: 1
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 0 }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 1 }}
{{- end }}
`,
			},
			wantErr: true,
		},
		{
			name: "should contain one element",
			args: args{
				actualData: `
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    value: 2
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 1 }}
{{- end }}
`,
			},
			wantErr: false,
		},
		{
			name: "fail to contain one element",
			args: args{
				actualData: `
metrics:
  - name: business-zone::projectA
    id: YnVzaW5lc3Mtem9uZTo6cHJvamVjdEE=.1
    value: 1
  - name: system::load balancer1
    id: c3lzdGVtOjpsb2FkIGJhbGFuY2VyMQ==.1
    value: 0
  - name: system::load balancer2
    id: WW91cl9BcHBsaWNhdGlvbk5hbWU=.1
    value: 2
`,
				expectedTemplate: `
metrics:
{{- contains .metrics }}
  - name: {{ notEmpty .name }}
    id: {{ notEmpty .id }}
    value: {{ gt .value 3 }}
{{- end }}
`,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := verify(tt.args.actualData, tt.args.expectedTemplate); (err != nil) != tt.wantErr {
				t.Errorf("verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
