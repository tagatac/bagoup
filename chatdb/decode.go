package chatdb

import (
	"fmt"
	"strings"

	ts "github.com/tagatac/typedstream-go"
)

func (d *chatDB) decodeTypedStream(s string) (string, error) {
	u, err := ts.NewUnarchiverFromData([]byte(s))
	if err != nil {
		return "", fmt.Errorf("create unarchiver: %w", err)
	}
	groups, err := u.DecodeAll()
	if err != nil {
		return "", fmt.Errorf("decode all: %w", err)
	}
	obj, err := extractArchivedObject(groups)
	if err != nil {
		return "", err
	}
	baseStr, err := extractBaseString(obj)
	if err != nil {
		return "", err
	}
	extras := collectExtras(obj)
	if len(extras) > 0 {
		return baseStr + "{\n" + strings.Join(extras, "\n") + "\n}", nil
	}
	return baseStr, nil
}

// extractArchivedObject pulls the top-level NSMutableAttributedString object
// out of the decoded typedstream groups.
func extractArchivedObject(groups []*ts.TypedGroup) (*ts.GenericArchivedObject, error) {
	if len(groups) == 0 || len(groups[0].Values) == 0 {
		return nil, fmt.Errorf("empty stream")
	}
	obj, ok := groups[0].Values[0].(*ts.GenericArchivedObject)
	if !ok {
		return nil, fmt.Errorf("unexpected top-level type %T", groups[0].Values[0])
	}
	return obj, nil
}

// extractBaseString pulls the NSMutableString / NSString value from the first
// content group of the archived object.
func extractBaseString(obj *ts.GenericArchivedObject) (string, error) {
	if len(obj.Contents) == 0 || len(obj.Contents[0].Values) == 0 {
		return "", fmt.Errorf("no string content")
	}
	switch v := obj.Contents[0].Values[0].(type) {
	case *ts.NSMutableString:
		return v.Value, nil
	case *ts.NSString:
		return v.Value, nil
	default:
		return "", fmt.Errorf("unexpected string type %T", obj.Contents[0].Values[0])
	}
}

// collectExtras scans remaining content groups for NSDictionary attributes,
// returning formatted key=value pairs for keys that don't start with "__kIM"
// (e.g. IMAudioTranscription). Format mirrors ObjC description style.
func collectExtras(obj *ts.GenericArchivedObject) []string {
	var extras []string
	for _, group := range obj.Contents[1:] {
		for _, val := range group.Values {
			dict, ok := val.(*ts.NSDictionary)
			if !ok {
				continue
			}
			for _, kv := range dict.Contents {
				k, ok := kv.Key.(*ts.NSString)
				if !ok || strings.HasPrefix(k.Value, "__kIM") {
					continue
				}
				if v, ok := kv.Value.(*ts.NSString); ok {
					extras = append(extras, fmt.Sprintf(`    %s = "%s"`, k.Value, v.Value))
				}
			}
		}
	}
	return extras
}
