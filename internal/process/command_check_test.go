package process

import (
	"strings"
	"testing"
)

func TestWindowsCommandProblems(t *testing.T) {
	env := map[string]string{
		"BERTH_DATA_DIR":  `C:\Users\me\project with spaces\.berth\data`,
		"BERTH_PORT_WEB":  "20000",
		"BERTH_WORKSPACE": `C:\Users\me\project with spaces\.berth\ws`,
	}
	cases := []struct {
		name      string
		processes map[string]any
		want      []string
	}{
		{
			name:      "clean command is accepted",
			processes: map[string]any{"web": map[string]any{"command": `python -m http.server ${BERTH_PORT_WEB} --bind 127.0.0.1`}},
		},
		{
			name:      "quoted argument is reported",
			processes: map[string]any{"web": map[string]any{"command": `python app.py --listen "127.0.0.1:${BERTH_PORT_WEB}"`}},
			want:      []string{"double quote"},
		},
		{
			name:      "path value with a space is reported",
			processes: map[string]any{"db": map[string]any{"command": `postgres -D ${BERTH_DATA_DIR}`}},
			want:      []string{"contains a space"},
		},
		{
			name:      "quoted path value reports both reasons",
			processes: map[string]any{"db": map[string]any{"command": `postgres -D "${BERTH_DATA_DIR}"`}},
			want:      []string{"double quote", "contains a space"},
		},
		{
			name:      "relative path avoids the problem",
			processes: map[string]any{"db": map[string]any{"command": `postgres -D .berth/data/pg`}},
		},
		{
			name:      "processes without a command are ignored",
			processes: map[string]any{"empty": map[string]any{"working_dir": "."}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WindowsCommandProblems(tc.processes, env)
			for _, want := range tc.want {
				if !containsSubstring(got, want) {
					t.Fatalf("problems = %v, want one mentioning %q", got, want)
				}
			}
			if len(tc.want) == 0 && len(got) != 0 {
				t.Fatalf("problems = %v, want none", got)
			}
		})
	}
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func TestWindowsCommandProblemsReportsEveryProcess(t *testing.T) {
	env := map[string]string{"BERTH_DATA_DIR": `/a b/c`}
	got := WindowsCommandProblems(map[string]any{
		"a": map[string]any{"command": `x "$BERTH_DATA_DIR"`},
		"b": map[string]any{"command": `y "$BERTH_DATA_DIR"`},
	}, env)
	if len(got) != 2 {
		t.Fatalf("problems = %v, want one entry per process", got)
	}
	if !strings.Contains(got[0], "processes.a") || !strings.Contains(got[1], "processes.b") {
		t.Fatalf("problems must name the processes in a stable order: %v", got)
	}
}
