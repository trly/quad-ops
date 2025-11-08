//go:build darwin

package launchd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// Plist represents a launchd property list.
type Plist struct {
	Label                string
	Program              string
	ProgramArguments     []string
	EnvironmentVariables map[string]string
	WorkingDirectory     string
	UserName             string
	GroupName            string
	RunAtLoad            bool
	KeepAlive            interface{} // bool or map[string]bool
	StandardOutPath      string
	StandardErrorPath    string
	ThrottleInterval     int
	AbandonProcessGroup  bool
	ProcessType          string
	SessionCreate        bool
	DependsOn            []string // Service dependencies (service labels)
}

// EncodePlist encodes a Plist to XML format.
func EncodePlist(p *Plist) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write XML header
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	buf.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	buf.WriteString("<plist version=\"1.0\">\n<dict>\n")

	// Write required fields
	writeDictEntry(buf, "Label", p.Label)

	// Write Program or ProgramArguments (mutually exclusive)
	if len(p.ProgramArguments) > 0 {
		writeDictArrayEntry(buf, "ProgramArguments", p.ProgramArguments)
	} else if p.Program != "" {
		writeDictEntry(buf, "Program", p.Program)
	}

	// Write optional fields
	if len(p.EnvironmentVariables) > 0 {
		writeDictDictEntry(buf, "EnvironmentVariables", p.EnvironmentVariables)
	}

	if p.WorkingDirectory != "" {
		writeDictEntry(buf, "WorkingDirectory", p.WorkingDirectory)
	}

	if p.UserName != "" {
		writeDictEntry(buf, "UserName", p.UserName)
	}

	if p.GroupName != "" {
		writeDictEntry(buf, "GroupName", p.GroupName)
	}

	writeDictBoolEntry(buf, "RunAtLoad", p.RunAtLoad)

	// Handle KeepAlive (can be bool or dict)
	if p.KeepAlive != nil {
		switch ka := p.KeepAlive.(type) {
		case bool:
			writeDictBoolEntry(buf, "KeepAlive", ka)
		case map[string]bool:
			buf.WriteString("\t<key>KeepAlive</key>\n\t<dict>\n")
			for k, v := range ka {
				writeDictBoolEntry(buf, k, v)
			}
			buf.WriteString("\t</dict>\n")
		}
	}

	if p.StandardOutPath != "" {
		writeDictEntry(buf, "StandardOutPath", p.StandardOutPath)
	}

	if p.StandardErrorPath != "" {
		writeDictEntry(buf, "StandardErrorPath", p.StandardErrorPath)
	}

	if p.ThrottleInterval > 0 {
		writeDictIntEntry(buf, "ThrottleInterval", p.ThrottleInterval)
	}

	writeDictBoolEntry(buf, "AbandonProcessGroup", p.AbandonProcessGroup)

	if p.ProcessType != "" {
		writeDictEntry(buf, "ProcessType", p.ProcessType)
	}

	if p.SessionCreate {
		writeDictBoolEntry(buf, "SessionCreate", p.SessionCreate)
	}

	if len(p.DependsOn) > 0 {
		writeDictArrayEntry(buf, "DependsOn", p.DependsOn)
	}

	// Close plist
	buf.WriteString("</dict>\n</plist>\n")

	return buf.Bytes(), nil
}

// writeDictEntry writes a string key-value entry.
func writeDictEntry(buf *bytes.Buffer, key, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(buf, "\t<key>%s</key>\n\t<string>%s</string>\n",
		xmlEscape(key), xmlEscape(value))
}

// writeDictBoolEntry writes a boolean key-value entry.
func writeDictBoolEntry(buf *bytes.Buffer, key string, value bool) {
	boolStr := "false"
	if value {
		boolStr = "true"
	}
	fmt.Fprintf(buf, "\t<key>%s</key>\n\t<%s/>\n", xmlEscape(key), boolStr)
}

// writeDictIntEntry writes an integer key-value entry.
func writeDictIntEntry(buf *bytes.Buffer, key string, value int) {
	if value == 0 {
		return
	}
	fmt.Fprintf(buf, "\t<key>%s</key>\n\t<integer>%d</integer>\n", xmlEscape(key), value)
}

// writeDictArrayEntry writes an array key-value entry.
func writeDictArrayEntry(buf *bytes.Buffer, key string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(buf, "\t<key>%s</key>\n\t<array>\n", xmlEscape(key))
	for _, v := range values {
		fmt.Fprintf(buf, "\t\t<string>%s</string>\n", xmlEscape(v))
	}
	buf.WriteString("\t</array>\n")
}

// writeDictDictEntry writes a dict key-value entry.
func writeDictDictEntry(buf *bytes.Buffer, key string, values map[string]string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(buf, "\t<key>%s</key>\n\t<dict>\n", xmlEscape(key))
	for k, v := range values {
		fmt.Fprintf(buf, "\t\t<key>%s</key>\n\t\t<string>%s</string>\n",
			xmlEscape(k), xmlEscape(v))
	}
	buf.WriteString("\t</dict>\n")
}

// xmlEscape escapes special XML characters.
func xmlEscape(s string) string {
	buf := new(bytes.Buffer)
	_ = xml.EscapeText(buf, []byte(s))
	return buf.String()
}

// SanitizeLabel sanitizes a string for use as a launchd label.
// Only allows: A-Z, a-z, 0-9, period, dash, underscore.
func SanitizeLabel(s string) string {
	var result strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}
	return result.String()
}
