package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel() // テストケースを並行して実行

	// テスト用の設定ファイルを作成
	createTestConfigFile := func(content string) (string, error) {
		tempFile, err := os.CreateTemp("", "config-*.yaml")
		if err != nil {
			return "", err
		}
		if _, err := tempFile.WriteString(content); err != nil {
			return "", err
		}
		if err := tempFile.Close(); err != nil {
			return "", err
		}
		return tempFile.Name(), nil
	}

	// テストケースの定義
	cases := []struct {
		name        string
		configData  string
		expected    Config
		expectedErr bool
	}{
		{
			name: "ValidConfig",
			configData: `
scriptPath:
  - /path/to/script1
  - /path/to/script2
maxAffectedNodeCount: 100
targetNodeLabels:
  app: web
`,
			expected: Config{
				ScriptPath:           []string{"/path/to/script1", "/path/to/script2"},
				MaxAffectedNodeCount: 100,
				TargetNodeLabels:     map[string]string{"app": "web"},
			},
			expectedErr: false,
		},
		{
			name: "InvalidConfig",
			configData: `
invalidYAMLContent
`,
			expected:    Config{},
			expectedErr: true,
		},
	}

	// テストケースを順に実行
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configFileName, err := createTestConfigFile(tc.configData)
			if err != nil {
				t.Fatalf("Failed to create config file: %v", err)
			}
			defer os.Remove(configFileName)

			actual, err := LoadConfig(configFileName)
			if (err != nil) != tc.expectedErr {
				t.Errorf("LoadConfig() error = %v, expectedErr %v", err, tc.expectedErr)
			}

			// エラーがなく、実際の結果と期待される結果を比較
			if err == nil && !reflect.DeepEqual(actual, &tc.expected) {
				t.Errorf("LoadConfig() = %v, want %v", actual, &tc.expected)
			}
		})
	}
}
