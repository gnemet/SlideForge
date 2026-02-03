package i18n

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"path/filepath"
)

var translations = make(map[string]map[string]string)

func Init(resourcesFS fs.FS) {
	files, _ := fs.ReadDir(resourcesFS, ".")
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			lang := f.Name()[:len(f.Name())-5]
			data, _ := fs.ReadFile(resourcesFS, f.Name())
			var t map[string]string
			json.Unmarshal(data, &t)
			translations[lang] = t
		}
	}
}

func T(lang, key string) string {
	if t, ok := translations[lang]; ok {
		if val, ok := t[key]; ok {
			return val
		}
	}
	// Fallback to en
	if t, ok := translations["en"]; ok {
		if val, ok := t[key]; ok {
			return val
		}
	}
	return key
}

func GetLang(r *http.Request) string {
	cookie, err := r.Cookie("lang")
	if err == nil {
		return cookie.Value
	}
	return "en"
}

func GetAvailableLangs() []string {
	langs := []string{}
	for l := range translations {
		langs = append(langs, l)
	}
	if len(langs) == 0 {
		return []string{"en", "hu"}
	}
	return langs
}
