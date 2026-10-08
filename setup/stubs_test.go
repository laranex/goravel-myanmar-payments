package main

import (
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigStub(t *testing.T) {
	tests := []struct {
		pkg, facades string
	}{
		{"config", "goravel/app/facades"},
		{"settings", "example.com/shop/app/facades"},
	}
	for _, tt := range tests {
		t.Run(tt.facades, func(t *testing.T) {
			stub := configStub(tt.pkg, tt.facades)
			file, err := parser.ParseFile(token.NewFileSet(), "myanmar_payments.go", stub, parser.ImportsOnly)
			require.NoError(t, err)
			assert.Equal(t, tt.pkg, file.Name.Name)
			require.Len(t, file.Imports, 2)
			assert.Equal(t, `"github.com/goravel/framework/contracts/http"`, file.Imports[0].Path.Value)
			assert.Equal(t, `"`+tt.facades+`"`, file.Imports[1].Path.Value)
			assert.NotContains(t, stub, "Dummy")
		})
	}
}

// TestConfigStubMatchesThePublishedConfig keeps the install stub and the file
// `vendor:publish` copies identical.
func TestConfigStubMatchesThePublishedConfig(t *testing.T) {
	published, err := os.ReadFile("../config/myanmar_payments.go")
	require.NoError(t, err)

	assert.Equal(t, string(published), configStub("config", "github.com/goravel/framework/facades"))
}

func TestEnvStubListsTheGatewayCredentials(t *testing.T) {
	published, err := os.ReadFile("../config/myanmar_payments.go")
	require.NoError(t, err)
	env := envStub()

	for _, match := range regexp.MustCompile(`config\.Env\("((?:KBZ_PAY|WAVE_MONEY|AYA_PAY|YOMA_MMQR|CYBER_SOURCE)_(?:SANDBOX|APP_ID|APP_KEY|APP_SECRET|MERCHANT_CODE|MERCHANT_ID|SECRET_KEY|MERCHANT_NAME|CLIENT_ID|CLIENT_SECRET|WEBHOOK_HASHKEY|WEBHOOK_SECRET|PROFILE_ID|ACCESS_KEY))"`).FindAllStringSubmatch(string(published), -1) {
		assert.Contains(t, env, "\n"+match[1]+"=", match[1])
	}
	for _, line := range strings.Split(strings.TrimSpace(env), "\n") {
		if line != "" {
			assert.Regexp(t, `^[A-Z_]+=(true)?$`, line)
		}
	}
}
