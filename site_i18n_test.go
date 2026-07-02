package gosherpa_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	i18nLanguagePattern = regexp.MustCompile(`^\s*([a-z]{2}):\s*\{\s*$`)
	i18nKeyPattern      = regexp.MustCompile(`^\s*"([^"]+)":`)
	htmlI18nPattern     = regexp.MustCompile(`data-i18n(?:-label|-alt)?="([^"]+)"`)
)

func TestSiteTranslationsHaveMatchingKeys(t *testing.T) {
	dictionaries := readSiteDictionaries(t)
	defaultDictionary := dictionaries["de"]
	if len(defaultDictionary) == 0 {
		t.Fatal("missing default de translation dictionary")
	}

	for language, dictionary := range dictionaries {
		if language == "de" {
			continue
		}

		missing := missingKeys(defaultDictionary, dictionary)
		extra := missingKeys(dictionary, defaultDictionary)
		if len(missing) > 0 || len(extra) > 0 {
			t.Fatalf("%s translations differ from de\nmissing: %s\nextra: %s", language, strings.Join(missing, ", "), strings.Join(extra, ", "))
		}
	}
}

func TestPageI18nKeysExistInDefaultDictionary(t *testing.T) {
	dictionaries := readSiteDictionaries(t)
	defaultDictionary := dictionaries["de"]
	if len(defaultDictionary) == 0 {
		t.Fatal("missing default de translation dictionary")
	}

	html, err := os.ReadFile("index.html")
	if err != nil {
		t.Fatal(err)
	}

	requiredKeys := map[string]bool{
		"meta.title":         true,
		"meta.description":   true,
		"meta.ogDescription": true,
		"meta.locale":        true,
	}
	for _, match := range htmlI18nPattern.FindAllStringSubmatch(string(html), -1) {
		requiredKeys[match[1]] = true
	}

	var missing []string
	for key := range requiredKeys {
		if !defaultDictionary[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)

	if len(missing) > 0 {
		t.Fatalf("missing default translations for HTML keys: %s", strings.Join(missing, ", "))
	}
}

func readSiteDictionaries(t *testing.T) map[string]map[string]bool {
	t.Helper()

	contents, err := os.ReadFile("assets/i18n.js")
	if err != nil {
		t.Fatal(err)
	}

	dictionaries := make(map[string]map[string]bool)
	var currentLanguage string

	for _, line := range strings.Split(string(contents), "\n") {
		if match := i18nLanguagePattern.FindStringSubmatch(line); match != nil {
			currentLanguage = match[1]
			dictionaries[currentLanguage] = make(map[string]bool)
			continue
		}

		if currentLanguage == "" {
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(line), "}") {
			currentLanguage = ""
			continue
		}

		if match := i18nKeyPattern.FindStringSubmatch(line); match != nil {
			dictionaries[currentLanguage][match[1]] = true
		}
	}

	if len(dictionaries) == 0 {
		t.Fatal("no translation dictionaries found in assets/i18n.js")
	}

	return dictionaries
}

func missingKeys(expected map[string]bool, actual map[string]bool) []string {
	var missing []string
	for key := range expected {
		if !actual[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}
