// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// CommentPrefix starts a key that is ignored at every level: how the owner comments (03).
const CommentPrefix = "//"

// stripComments returns raw with every object key beginning with "//" removed, at every level,
// re-encoded compactly with object keys sorted and HTML characters left as they are. It is
// also the syntax check: invalid JSON or trailing data is an error.
func stripComments(raw []byte) ([]byte, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, errors.New("empty file")
	}
	switch raw[0] {
	case '{':
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		for k, v := range m {
			if strings.HasPrefix(k, CommentPrefix) {
				delete(m, k)
				continue
			}
			s, err := stripComments(v)
			if err != nil {
				return nil, err
			}
			m[k] = s
		}
		return encode(m)
	case '[':
		var a []json.RawMessage
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, err
		}
		for i, v := range a {
			s, err := stripComments(v)
			if err != nil {
				return nil, err
			}
			a[i] = s
		}
		return encode(a)
	}
	if !json.Valid(raw) {
		var v json.RawMessage
		return nil, json.Unmarshal(raw, &v)
	}
	return raw, nil
}

// encode is json.Marshal without HTML escaping, so "<root>" prints as written.
func encode[T map[string]json.RawMessage | []json.RawMessage](v T) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b.Bytes()), nil
}

// syntaxMessage turns a JSON syntax error into "line L, column C: <what>" against src.
func syntaxMessage(src []byte, err error) string {
	var se *json.SyntaxError
	if !errors.As(err, &se) {
		return err.Error()
	}
	line, col := 1, 1
	for i := 0; i < int(se.Offset)-1 && i < len(src); i++ {
		if src[i] == '\n' {
			line, col = line+1, 1
		} else {
			col++
		}
	}
	return fmt.Sprintf("invalid JSON at line %d, column %d: %s", line, col, se.Error())
}

// unknownKeys lists, sorted, the dotted path of every key in raw that t does not declare,
// matched exactly (encoding/json alone would match case-insensitively and stop at the first).
// A value of the wrong JSON type is left to the decoder to report.
func unknownKeys(raw json.RawMessage, t reflect.Type, path string) []string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	raw = bytes.TrimSpace(raw)
	var out []string
	// The config types use only struct, map and slice containers; every other kind is a leaf.
	//exhaustive:ignore
	switch t.Kind() {
	case reflect.Struct:
		var m map[string]json.RawMessage
		if len(raw) == 0 || raw[0] != '{' || json.Unmarshal(raw, &m) != nil {
			return nil
		}
		fields := map[string]reflect.Type{}
		for i := range t.NumField() {
			f := t.Field(i)
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name != "" && name != "-" {
				fields[name] = f.Type
			}
		}
		for _, k := range sortedKeys(m) {
			ft, ok := fields[k]
			if !ok {
				out = append(out, join(path, k))
				continue
			}
			out = append(out, unknownKeys(m[k], ft, join(path, k))...)
		}
	case reflect.Map:
		var m map[string]json.RawMessage
		if len(raw) == 0 || raw[0] != '{' || json.Unmarshal(raw, &m) != nil {
			return nil
		}
		for _, k := range sortedKeys(m) {
			out = append(out, unknownKeys(m[k], t.Elem(), join(path, k))...)
		}
	case reflect.Slice:
		var a []json.RawMessage
		if len(raw) == 0 || raw[0] != '[' || json.Unmarshal(raw, &a) != nil {
			return nil
		}
		for i, v := range a {
			out = append(out, unknownKeys(v, t.Elem(), join(path, strconv.Itoa(i)))...)
		}
	default:
	}
	return out
}

func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// decodeFile parses one configuration file into T: comments stripped, every unknown key
// reported, then a strict decode (DisallowUnknownFields, set here and nowhere else in the
// module) for the types. It returns the stripped form and the problems, one message each.
func decodeFile[T Instance | Project](src []byte) (T, []byte, []string) {
	var v T
	src = trimBOM(src)
	if len(bytes.TrimSpace(src)) == 0 {
		return v, nil, []string{"empty file"}
	}
	// The syntax check runs on the bytes as read, so a syntax error's line and column are the file's.
	var probe json.RawMessage
	if err := json.Unmarshal(src, &probe); err != nil {
		return v, nil, []string{syntaxMessage(src, err)}
	}
	stripped, err := stripComments(src)
	if err != nil {
		return v, nil, []string{err.Error()}
	}
	if stripped[0] != '{' {
		return v, nil, []string{"want a JSON object"}
	}
	var msgs []string
	for _, k := range duplicateKeys(src) {
		msgs = append(msgs, "duplicate key "+k)
	}
	for _, k := range unknownKeys(stripped, reflect.TypeFor[T](), "") {
		msgs = append(msgs, "unknown key "+k)
	}
	if len(msgs) > 0 {
		return v, nil, msgs
	}
	dec := json.NewDecoder(bytes.NewReader(stripped))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, nil, []string{typeMessage(err)}
	}
	return v, stripped, nil
}

// typeMessage turns a decode error into "<key path>: want <type>, got <JSON type>".
func typeMessage(err error) string {
	var te *json.UnmarshalTypeError
	if errors.As(err, &te) {
		field := te.Field
		if field == "" {
			field = "value"
		}
		return fmt.Sprintf("%s: want %s, got %s", field, goTypeName(te.Type), te.Value)
	}
	return err.Error()
}

func goTypeName(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// The kinds the config types use; anything else prints its Go name.
	//exhaustive:ignore
	switch t.Kind() {
	case reflect.String:
		return "a string"
	case reflect.Int, reflect.Int64, reflect.Int32:
		return "an integer"
	case reflect.Bool:
		return "true or false"
	case reflect.Slice:
		return "an array"
	case reflect.Map, reflect.Struct:
		return "an object"
	default:
		return t.String()
	}
}

// utf8BOM is what PowerShell 5.1's `-Encoding UTF8` puts before the text.
var utf8BOM = []byte("\xef\xbb\xbf")

func trimBOM(src []byte) []byte { return bytes.TrimPrefix(src, utf8BOM) }

// duplicateKeys lists the dotted path of every key that appears twice in one object of src,
// valid JSON, in file order. Comment keys may repeat ("//" on several lines is the usual way
// to write a long comment) and their values are not walked. Decoding into a map keeps only the
// last of two equal keys, so without this the first is lost silently.
func duplicateKeys(src []byte) []string {
	dec := json.NewDecoder(bytes.NewReader(src))
	var dups []string
	var walk func(path string) error
	walk = func(path string) error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch tok {
		case json.Delim('{'):
			seen := map[string]bool{}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return err
				}
				key, _ := kt.(string)
				p := join(path, key)
				if strings.HasPrefix(key, CommentPrefix) {
					var skip json.RawMessage
					if err := dec.Decode(&skip); err != nil {
						return err
					}
					continue
				}
				if seen[key] {
					dups = append(dups, p)
				}
				seen[key] = true
				if err := walk(p); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		case json.Delim('['):
			for i := 0; dec.More(); i++ {
				if err := walk(join(path, strconv.Itoa(i))); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		}
		return nil
	}
	if walk("") != nil {
		return nil // the syntax check has already reported it
	}
	return dups
}
