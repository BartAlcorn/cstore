package token

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func replaceJSON(b []byte, tokens map[string]Token, formattedValue bool) (d []byte, err error) {

	json := string(b)

	for _, t := range tokens {
		path := strings.Replace(t.EnvVar, "/", ".", -1)

		result := gjson.Get(json, path)

		if result.Exists() {
			b := bytes.Replace([]byte(result.Str), []byte(t.Formatted()), []byte(t.GetValue(formattedValue)), -1)

			json, err = sjson.Set(json, path, string(b))
			if err != nil {
				return []byte(json), err
			}
		}
	}

	return []byte(json), err
}

func searchJSON(b []byte, withValues bool) (tokens map[string]Token, err error) {
	tokens = map[string]Token{}

	var f interface{}
	if err := json.Unmarshal(b, &f); err != nil {
		return tokens, err
	}

	return getPropTokens(f, "", withValues), nil
}

func getPropTokens(f interface{}, root string, withValues bool) map[string]Token {
	m := f.(map[string]interface{})

	tokens := map[string]Token{}

	for path, v := range m {
		if len(root) > 0 {
			path = fmt.Sprintf("%s/%s", root, path)
		}

		switch vv := v.(type) {
		case map[string]interface{}:
			for k, v := range getPropTokens(vv, path, withValues) {
				tokens[k] = v
			}
		case []interface{}:
			for i, uu := range vv {

				indexedPath := fmt.Sprintf("%s/%d", path, i)

				switch uu.(type) {
				case string:
					for k, v := range getValueTokens(indexedPath, uu.(string), withValues) {
						tokens[k] = v
					}
				case map[string]interface{}:
					for k, v := range getPropTokens(uu, indexedPath, withValues) {
						tokens[k] = v
					}
				}
			}
		case string:
			for k, v := range getValueTokens(path, v.(string), withValues) {
				tokens[k] = v
			}
		}

	}

	return tokens
}

func getValueTokens(key, value string, forValues bool) map[string]Token {

	var tokenRegex = regexp.MustCompile(getTokenRegex(forValues))

	byteTokens := tokenRegex.FindAllSubmatch([]byte(value), -1)

	if byteTokens == nil {
		return map[string]Token{}
	}

	tokens := map[string]Token{}
	for _, bt := range byteTokens {

		ss := strings.Split(string(bt[2]), "/")
		nss := ss[:len(ss)-1]

		t := Token{
			EnvVar: key,
			Env:    strings.Join(nss, "/"),
			Prop:   ss[len(ss)-1],
			Value:  notFound,
		}

		if len(bt) == 4 {
			t.Value = string(bt[3])
		}

		tokens[t.String()] = t
	}

	return tokens
}
