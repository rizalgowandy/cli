package configure_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/fastly/cli/pkg/app"
	"github.com/fastly/cli/pkg/config"
	"github.com/fastly/cli/pkg/mock"
	"github.com/fastly/cli/pkg/testutil"
	"github.com/fastly/cli/pkg/update"
	"github.com/fastly/go-fastly/v3/fastly"
)

func TestConfigure(t *testing.T) {
	var (
		goodToken = func() (*fastly.Token, error) { return &fastly.Token{}, nil }
		badToken  = func() (*fastly.Token, error) { return nil, errors.New("bad token") }
		goodUser  = func(*fastly.GetUserInput) (*fastly.User, error) {
			return &fastly.User{
				Login: "test@example.com",
			}, nil
		}
		badUser = func(*fastly.GetUserInput) (*fastly.User, error) { return nil, errors.New("bad user") }
	)

	for _, testcase := range []struct {
		name           string
		args           []string
		env            config.Environment
		file           config.File
		api            mock.API
		configFileData string
		stdin          string
		wantError      string
		wantOutput     []string
		wantFile       string
	}{
		{
			name: "endpoint from flag",
			args: []string{"configure", "--endpoint=http://local.dev", "--token=abcdef"},
			api: mock.API{
				GetTokenSelfFn: goodToken,
				GetUserFn:      goodUser,
			},
			wantOutput: []string{
				"Fastly API endpoint (via --endpoint): http://local.dev",
				"Fastly API token provided via --token",
				"Validating token...",
				"Persisting configuration...",
				"Configured the Fastly CLI",
				"You can find your configuration file at",
			},
			wantFile: `
[cli]
  last_checked = ""
  remote_config = ""
  ttl = ""
  version = ""

[fastly]
  api_endpoint = "http://local.dev"

[language]

  [language.rust]
    fastly_sys_constraint = ""
    rustup_constraint = ""
    toolchain_version = ""
    wasm_wasi_target = ""

[legacy]
  email = ""
  token = ""

[starter-kits]

[user]
  email = "test@example.com"
  token = "abcdef"
`,
		},
		{
			name:           "endpoint already in file should be replaced by flag",
			args:           []string{"configure", "--endpoint=http://staging.dev", "--token=abcdef"},
			configFileData: "endpoint = \"https://api.fastly.com\"",
			stdin:          "new_token\n",
			api: mock.API{
				GetTokenSelfFn: goodToken,
				GetUserFn:      goodUser,
			},
			wantOutput: []string{
				"Fastly API endpoint (via --endpoint): http://staging.dev",
				"Fastly API token provided via --token",
				"Validating token...",
				"Persisting configuration...",
				"Configured the Fastly CLI",
				"You can find your configuration file at",
			},
			wantFile: `
[cli]
  last_checked = ""
  remote_config = ""
  ttl = ""
  version = ""

[fastly]
  api_endpoint = "http://staging.dev"

[language]

  [language.rust]
    fastly_sys_constraint = ""
    rustup_constraint = ""
    toolchain_version = ""
    wasm_wasi_target = ""

[legacy]
  email = ""
  token = ""

[starter-kits]

[user]
  email = "test@example.com"
  token = "abcdef"
`,
		},
		{
			name: "token from flag",
			args: []string{"configure", "--token=abcdef"},
			api: mock.API{
				GetTokenSelfFn: goodToken,
				GetUserFn:      goodUser,
			},
			wantOutput: []string{
				"Fastly API token provided via --token",
				"Validating token...",
				"Persisting configuration...",
				"Configured the Fastly CLI",
				"You can find your configuration file at",
			},
			wantFile: `
[cli]
  last_checked = ""
  remote_config = ""
  ttl = ""
  version = ""

[fastly]
  api_endpoint = "https://api.fastly.com"

[language]

  [language.rust]
    fastly_sys_constraint = ""
    rustup_constraint = ""
    toolchain_version = ""
    wasm_wasi_target = ""

[legacy]
  email = ""
  token = ""

[starter-kits]

[user]
  email = "test@example.com"
  token = "abcdef"
`,
		},
		{
			name:  "token from interactive input",
			args:  []string{"configure"},
			stdin: "1234\n",
			api: mock.API{
				GetTokenSelfFn: goodToken,
				GetUserFn:      goodUser,
			},
			wantOutput: []string{
				"An API token is used to authenticate requests to the Fastly API. To create a token, visit",
				"https://manage.fastly.com/account/personal/tokens",
				"Fastly API token: ",
				"Validating token...",
				"Persisting configuration...",
				"Configured the Fastly CLI",
				"You can find your configuration file at",
			},
			wantFile: `
[cli]
  last_checked = ""
  remote_config = ""
  ttl = ""
  version = ""

[fastly]
  api_endpoint = "https://api.fastly.com"

[language]

  [language.rust]
    fastly_sys_constraint = ""
    rustup_constraint = ""
    toolchain_version = ""
    wasm_wasi_target = ""

[legacy]
  email = ""
  token = ""

[starter-kits]

[user]
  email = "test@example.com"
  token = "1234"
`,
		},
		{
			name: "token from environment",
			args: []string{"configure"},
			env:  config.Environment{Token: "hello"},
			api: mock.API{
				GetTokenSelfFn: goodToken,
				GetUserFn:      goodUser,
			},
			wantOutput: []string{
				"Fastly API token provided via FASTLY_API_TOKEN",
				"Validating token...",
				"Persisting configuration...",
				"Configured the Fastly CLI",
				"You can find your configuration file at",
			},
			wantFile: `
[cli]
  last_checked = ""
  remote_config = ""
  ttl = ""
  version = ""

[fastly]
  api_endpoint = "https://api.fastly.com"

[language]

  [language.rust]
    fastly_sys_constraint = ""
    rustup_constraint = ""
    toolchain_version = ""
    wasm_wasi_target = ""

[legacy]
  email = ""
  token = ""

[starter-kits]

[user]
  email = "test@example.com"
  token = "hello"
`,
		},
		{
			name:           "token already in file should trigger interactive input",
			args:           []string{"configure"},
			configFileData: "token = \"old_token\"",
			stdin:          "new_token\n",
			api: mock.API{
				GetTokenSelfFn: goodToken,
				GetUserFn:      goodUser,
			},
			wantOutput: []string{
				"An API token is used to authenticate requests to the Fastly API. To create a token, visit",
				"https://manage.fastly.com/account/personal/tokens",
				"Fastly API token: ",
				"Validating token...",
				"Persisting configuration...",
				"Configured the Fastly CLI",
				"You can find your configuration file at",
			},
			wantFile: `
[cli]
  last_checked = ""
  remote_config = ""
  ttl = ""
  version = ""

[fastly]
  api_endpoint = "https://api.fastly.com"

[language]

  [language.rust]
    fastly_sys_constraint = ""
    rustup_constraint = ""
    toolchain_version = ""
    wasm_wasi_target = ""

[legacy]
  email = ""
  token = ""

[starter-kits]

[user]
  email = "test@example.com"
  token = "new_token"
`,
		},
		{
			name: "invalid token",
			args: []string{"configure", "--token=abcdef"},
			api: mock.API{
				GetTokenSelfFn: badToken,
				GetUserFn:      badUser,
			},
			wantOutput: []string{
				"Fastly API token provided via --token",
				"Validating token...",
			},
			wantError: "error validating token: bad token",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			configFilePath := testutil.MakeTempFile(t, testcase.configFileData)
			defer os.RemoveAll(configFilePath)

			var (
				args                           = testcase.args
				env                            = testcase.env
				file                           = testcase.file
				clientFactory                  = mock.APIClient(testcase.api)
				httpClient                     = http.DefaultClient
				cliVersioner  update.Versioner = nil
				in            io.Reader        = strings.NewReader(testcase.stdin)
				out           bytes.Buffer
			)
			err := app.Run(args, env, file, configFilePath, clientFactory, httpClient, cliVersioner, in, &out)
			testutil.AssertErrorContains(t, err, testcase.wantError)
			for _, s := range testcase.wantOutput {
				testutil.AssertStringContains(t, out.String(), s)
			}
			if testcase.wantError == "" {
				p, err := os.ReadFile(configFilePath)
				testutil.AssertNoError(t, err)
				testutil.AssertString(t, testcase.wantFile, string(p))
			}
		})
	}
}
