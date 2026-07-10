package tui

import "testing"

func TestCommandNameFromText(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{name: "plain executable", command: "rg --files", want: "rg"},
		{name: "environment assignment", command: "MODE=test npm run build", want: "npm"},
		{name: "sudo option value", command: "sudo -u agent rg --files", want: "rg"},
		{name: "env wrapper", command: "env -i MODE=test node app.js", want: "node"},
		{name: "posix nested shell", command: `bash -lc "cd src && go test ./..."`, want: "go"},
		{name: "cmd nested shell", command: `cmd /c "where.exe go"`, want: "where"},
		{name: "powershell nested shell", command: `pwsh -Command "Get-ChildItem -Force"`, want: "get-childitem"},
		{name: "skip navigation segment", command: "cd src && git status", want: "git"},
		{name: "executable path", command: `C:\\Tools\\python.exe script.py`, want: "python"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandNameFromText(tt.command, 0); got != tt.want {
				t.Fatalf("commandNameFromText(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}
