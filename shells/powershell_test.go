//go:build !integration

package shells

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

func TestPowershell_LineBreaks(t *testing.T) {
	testCases := map[string]struct {
		shell                   string
		eol                     string
		expectedErrorPreference string
		shebang                 string
	}{
		"Windows newline on Desktop": {
			shell:                   SNPowershell,
			eol:                     "\r\n",
			expectedErrorPreference: "",
		},
		"Windows newline on Core": {
			shell:                   SNPwsh,
			eol:                     "\r\n",
			expectedErrorPreference: `$ErrorActionPreference = "Stop"` + "\r\n",
		},
		"Linux newline on Core": {
			shell:                   SNPwsh,
			eol:                     "\n",
			shebang:                 `#!/usr/bin/env pwsh` + "\n",
			expectedErrorPreference: `$ErrorActionPreference = "Stop"` + "\n",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			eol := tc.eol
			writer := &PsWriter{Shell: tc.shell, EOL: eol}
			writer.Command("foo", "")

			expectedOutput :=
				tc.expectedErrorPreference +
					`& "foo" ''` + eol + "if(!$?) { Exit &{if($LASTEXITCODE) {$LASTEXITCODE} else {1}} }" + eol +
					eol +
					eol
			if tc.shell == SNPwsh {
				expectedOutput = tc.shebang + "& {" + eol + eol + expectedOutput + "}" + eol + eol
			} else {
				expectedOutput = "\xef\xbb\xbf" + expectedOutput
			}
			assert.Equal(t, expectedOutput, writer.Finish(false))
		})
	}
}

func TestPowershell_CommandShellEscapes(t *testing.T) {
	writer := &PsWriter{Shell: SNPowershell, EOL: "\r\n"}
	writer.Command("foo", "x&(y)")

	assert.Equal(
		t,
		"& \"foo\" 'x&(y)'\r\nif(!$?) { Exit &{if($LASTEXITCODE) {$LASTEXITCODE} else {1}} }\r\n\r\n",
		writer.String(),
	)
}

func TestPowershell_IfCmdShellEscapes(t *testing.T) {
	writer := &PsWriter{Shell: SNPowershell, EOL: "\r\n"}
	writer.IfCmd("foo", "x&(y)")

	//nolint:lll
	assert.Equal(t, "Set-Variable -Name cmdErr -Value $false\r\nTry {\r\n  & \"foo\" 'x&(y)' 2>$null\r\n  if(!$?) { throw &{if($LASTEXITCODE) {$LASTEXITCODE} else {1}} }\r\n} Catch {\r\n  Set-Variable -Name cmdErr -Value $true\r\n}\r\nif(!$cmdErr) {\r\n", writer.String())
}

func TestPowershell_MkTmpDirOnUNCShare(t *testing.T) {
	writer := &PsWriter{TemporaryPath: `\\unc-server\share`, EOL: "\n"}
	writer.MkTmpDir("tmp")

	assert.Equal(
		t,
		`New-Item -ItemType directory -Force -Path "\\unc-server\share\tmp" | out-null`+writer.EOL,
		writer.String(),
	)
}

func TestPowershell_GetName(t *testing.T) {
	for _, shellName := range []string{SNPwsh, SNPowershell} {
		shell := common.GetShell(shellName)
		assert.Equal(t, shellName, shell.GetName())
	}
}

func TestPowershell_IsDefault(t *testing.T) {
	for _, shellName := range []string{SNPwsh, SNPowershell} {
		shell := common.GetShell(shellName)
		assert.False(t, shell.IsDefault())
	}
}

//nolint:lll
func TestPowershell_GetConfiguration(t *testing.T) {
	testCases := map[string]struct {
		shell    string
		executor string
		user     string
		os       string

		expectedError        error
		expectedPassFile     bool
		expectedCommand      string
		expectedCmdLine      string
		getExpectedArguments func() []string
	}{
		"powershell on docker-windows": {
			shell:    SNPowershell,
			executor: dockerWindowsExecutor,

			expectedPassFile:     false,
			expectedCommand:      SNPowershell,
			getExpectedArguments: stdinCmdArgs,
			expectedCmdLine:      "powershell -NoProfile -NoLogo -InputFormat text -OutputFormat text -NonInteractive -ExecutionPolicy Bypass -Command -",
		},
		"pwsh on docker-windows": {
			shell:    SNPwsh,
			executor: dockerWindowsExecutor,

			expectedPassFile:     false,
			expectedCommand:      SNPwsh,
			getExpectedArguments: stdinCmdArgs,
			expectedCmdLine:      "pwsh -NoProfile -NoLogo -InputFormat text -OutputFormat text -NonInteractive -ExecutionPolicy Bypass -Command -",
		},
		"pwsh on docker": {
			shell:    SNPwsh,
			executor: "docker",

			expectedPassFile:     false,
			expectedCommand:      SNPwsh,
			getExpectedArguments: stdinCmdArgs,
			expectedCmdLine:      "pwsh -NoProfile -NoLogo -InputFormat text -OutputFormat text -NonInteractive -ExecutionPolicy Bypass -Command -",
		},
		"pwsh on kubernetes": {
			shell:    SNPwsh,
			executor: "kubernetes",

			expectedPassFile:     false,
			expectedCommand:      SNPwsh,
			getExpectedArguments: stdinCmdArgs,
			expectedCmdLine:      "pwsh -NoProfile -NoLogo -InputFormat text -OutputFormat text -NonInteractive -ExecutionPolicy Bypass -Command -",
		},
		"pwsh on shell": {
			shell:    SNPwsh,
			executor: "shell",

			expectedPassFile:     false,
			expectedCommand:      SNPwsh,
			getExpectedArguments: stdinCmdArgs,
			expectedCmdLine:      "pwsh -NoProfile -NoLogo -InputFormat text -OutputFormat text -NonInteractive -ExecutionPolicy Bypass -Command -",
		},
		"powershell on shell": {
			shell:    SNPowershell,
			executor: "shell",

			expectedPassFile:     true,
			expectedCommand:      SNPowershell,
			getExpectedArguments: fileCmdArgs,
			expectedCmdLine:      "powershell -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command",
		},
		"pwsh on shell with custom user (linux)": {
			shell:    SNPwsh,
			executor: "shell",
			user:     "custom",
			os:       OSLinux,

			expectedPassFile: false,
			expectedCommand:  "su",
			expectedCmdLine:  "su -s /usr/bin/pwsh custom -c pwsh -NoProfile -NoLogo -InputFormat text -OutputFormat text -NonInteractive -ExecutionPolicy Bypass -Command -",
			getExpectedArguments: func() []string {
				return []string{"-s", "/usr/bin/" + SNPwsh, "custom", "-c", SNPwsh + " " + strings.Join(stdinCmdArgs(), " ")}
			},
		},
		"pwsh on shell with custom user (windows)": {
			shell:    SNPwsh,
			executor: "shell",
			user:     "custom",
			os:       OSWindows,

			expectedPassFile: false,
			expectedCommand:  "su",
			expectedCmdLine:  "su custom -c pwsh -NoProfile -NoLogo -InputFormat text -OutputFormat text -NonInteractive -ExecutionPolicy Bypass -Command -",
			getExpectedArguments: func() []string {
				return []string{"-s", "custom", "-c", SNPwsh + " " + strings.Join(stdinCmdArgs(), " ")}
			},
		},
		"powershell docker-windows change user": {
			shell:    SNPowershell,
			executor: "anything-but-docker-windows",
			user:     "custom",

			expectedError: &powershellChangeUserError{
				shell:    SNPowershell,
				executor: "anything-but-docker-windows",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			if tc.os != "" && tc.os != runtime.GOOS {
				t.Skipf("test only runs on %s", tc.os)
			}

			shell := common.GetShell(tc.shell)
			info := common.ShellScriptInfo{
				Shell: tc.shell,
				User:  tc.user,
				Build: &common.Build{
					Runner: &common.RunnerConfig{},
				},
			}
			info.Build.Runner.Executor = tc.executor

			shellConfig, err := shell.GetConfiguration(info)
			if tc.expectedError != nil {
				require.Equal(t, tc.expectedError, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.getExpectedArguments(), shellConfig.Arguments)
			assert.Equal(t, tc.expectedCommand, shellConfig.Command)
			assert.Equal(t, PowershellDockerCmd(tc.shell), shellConfig.DockerCommand)
			assert.Equal(t, tc.expectedCmdLine, shellConfig.CmdLine)
			assert.Equal(t, tc.expectedPassFile, shellConfig.PassFile)
			assert.Equal(t, "ps1", shellConfig.Extension)
		})
	}
}

func TestPowershellCmdArgs(t *testing.T) {
	for _, tc := range []string{SNPwsh, SNPowershell} {
		t.Run(tc, func(t *testing.T) {
			args := PowershellDockerCmd(tc)
			assert.Equal(t, append([]string{tc}, stdinCmdArgs()...), args)
		})
	}
}

//nolint:lll
func TestPowershellPathResolveOperations(t *testing.T) {
	var templateReplacer = func(escaped string) func(string) string {
		return func(tpl string) string {
			return fmt.Sprintf(tpl, escaped)
		}
	}

	testCases := map[string]struct {
		op       func(path string, w *PsWriter)
		template string
		expected map[string]func(string) string
	}{
		"cd": {
			op: func(path string, w *PsWriter) {
				w.Cd(path)
			},
			template: "cd $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v)\nif(!$?) { Exit &{if($LASTEXITCODE) {$LASTEXITCODE} else {1}} }\n\n",
			expected: map[string]func(string) string{
				`path/name`: templateReplacer(`"path/name"`),
				`\\unc\`:    templateReplacer(`"\\unc\"`),
				`C:\path\`:  templateReplacer(`"C:\path\"`),
			},
		},
		"mkdir": {
			op: func(path string, w *PsWriter) {
				w.MkDir(path)
			},
			template: "New-Item -ItemType directory -Force -Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v) | out-null\n",
			expected: map[string]func(string) string{
				`path/name`: templateReplacer(`"path/name"`),
				`\\unc\`:    templateReplacer(`"\\unc\"`),
				`C:\path\`:  templateReplacer(`"C:\path\"`),
			},
		},
		"mktmpdir": {
			op: func(path string, w *PsWriter) {
				w.TemporaryPath = path
				w.MkTmpDir("dir")
			},
			template: "New-Item -ItemType directory -Force -Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v) | out-null\n",
			expected: map[string]func(string) string{
				`path/name`: templateReplacer(`"path/name/dir"`),
				`\\unc\`:    templateReplacer(`"\\unc\/dir"`),
				`C:\path\`:  templateReplacer(`"C:\path\/dir"`),
			},
		},
		"rm": {
			op: func(path string, w *PsWriter) {
				w.RmFile(path)
			},
			template: "if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v) -PathType Leaf) ) {\n  Remove-Item2 -Force $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v)\n} elseif(Test-Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v)) {\n  Remove-Item -Force $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v)\n}\n\n",
			expected: map[string]func(string) string{
				`path/name`:    templateReplacer(`"path/name"`),
				`\\unc\file`:   templateReplacer(`"\\unc\file"`),
				`C:\path\file`: templateReplacer(`"C:\path\file"`),
			},
		},
		"rmfilesrecursive": {
			op: func(path string, w *PsWriter) {
				w.RmFilesRecursive(path, "test")
			},
			template: "if(Test-Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v) -PathType Container) {\n  Get-ChildItem -Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v) -Filter \"test\" -Recurse | ForEach-Object { Remove-Item -Force $_.FullName }\n}\n",
			expected: map[string]func(string) string{
				`path/name`:    templateReplacer(`"path/name"`),
				`\\unc\file`:   templateReplacer(`"\\unc\file"`),
				`C:\path\file`: templateReplacer(`"C:\path\file"`),
			},
		},
		"rmdir": {
			op: func(path string, w *PsWriter) {
				w.RmDir(path)
			},
			template: "if( (Get-Command -Name Remove-Item2 -Module NTFSSecurity -ErrorAction SilentlyContinue) -and (Test-Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v) -PathType Container) ) {\n  Remove-Item2 -Force -Recurse $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v)\n} elseif(Test-Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v)) {\n  Remove-Item -Force -Recurse $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v)\n}\n\n",
			expected: map[string]func(string) string{
				`path/name`:    templateReplacer(`"path/name"`),
				`\\unc\file`:   templateReplacer(`"\\unc\file"`),
				`C:\path\file`: templateReplacer(`"C:\path\file"`),
			},
		},
		"ifdirectory": {
			op: func(path string, w *PsWriter) {
				w.IfDirectory(path)
			},
			template: "if(Test-Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v) -PathType Container) {\n",
			expected: map[string]func(string) string{
				`path/name`:    templateReplacer(`"path/name"`),
				`\\unc\file`:   templateReplacer(`"\\unc\file"`),
				`C:\path\file`: templateReplacer(`"C:\path\file"`),
			},
		},
		"iffile": {
			op: func(path string, w *PsWriter) {
				w.IfFile(path)
			},
			template: "if(Test-Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(%[1]v) -PathType Leaf) {\n",
			expected: map[string]func(string) string{
				`path/name`:    templateReplacer(`"path/name"`),
				`\\unc\file`:   templateReplacer(`"\\unc\file"`),
				`C:\path\file`: templateReplacer(`"C:\path\file"`),
			},
		},
		"file variable": {
			op: func(path string, w *PsWriter) {
				w.TemporaryPath = path
				w.Variable(common.JobVariable{File: true, Key: "a key", Value: "foobar"})
			},
			template: "New-Item -ItemType directory -Force -Path $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(\"%[1]v\") | out-null\n[System.IO.File]::WriteAllText($ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(\"%[1]v/a key\"), \"foobar\")\n$a key=$ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath(\"%[1]v/a key\")\n$env:a key=$a key\n",
			expected: map[string]func(string) string{
				`path/name`:    templateReplacer(`path/name`),
				`\\unc\file`:   templateReplacer(`\\unc\file`),
				`C:\path\file`: templateReplacer(`C:\path\file`),
			},
		},
	}

	for tn, tc := range testCases {
		for path, expected := range tc.expected {
			for _, shell := range []string{SNPowershell, SNPwsh} {
				t.Run(fmt.Sprintf("%s:%s: %s", shell, tn, path), func(t *testing.T) {
					writer := &PsWriter{TemporaryPath: "\\temp", Shell: shell, EOL: "\n", resolvePaths: true}
					tc.op(path, writer)
					assert.Equal(t, expected(tc.template), writer.String())
				})
			}
		}
	}
}

func TestPowershell_GenerateScript(t *testing.T) {
	shellInfo := common.ShellScriptInfo{
		Shell:         "pwsh",
		Type:          common.NormalShell,
		RunnerCommand: "/usr/bin/gitlab-runner-helper",
		Build: &common.Build{
			Runner: &common.RunnerConfig{},
		},
	}
	shellInfo.Build.Runner.Executor = "kubernetes"
	shellInfo.Build.Hostname = "Test Hostname"

	pwshShell := common.GetShell("pwsh").(*PowerShell)
	shebang := ""
	if pwshShell.EOL == "\n" {
		shebang = "#!/usr/bin/env pwsh\n"
	}

	testCases := map[string]struct {
		stage           common.BuildStage
		info            common.ShellScriptInfo
		expectedFailure bool
		expectedScript  string
	}{
		"prepare script": {
			stage:           common.BuildStagePrepare,
			info:            shellInfo,
			expectedFailure: false,
			expectedScript: shebang + "& {" +
				pwshShell.EOL + pwshShell.EOL +
				`$ErrorActionPreference = "Stop"` + pwshShell.EOL +
				`echo "Running on $([Environment]::MachineName) via "Test Hostname"..."` +
				pwshShell.EOL + pwshShell.EOL + "}" + pwshShell.EOL + pwshShell.EOL,
		},
		"cleanup variables": {
			stage:           common.BuildStageCleanup,
			info:            shellInfo,
			expectedFailure: false,
			expectedScript:  ``,
		},
		"no script": {
			stage:           "no_script",
			info:            shellInfo,
			expectedFailure: true,
			expectedScript:  "",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			script, err := pwshShell.GenerateScript(tc.stage, tc.info)
			assert.Equal(t, tc.expectedScript, script)
			if tc.expectedFailure {
				assert.Error(t, err)
			}
		})
	}
}
