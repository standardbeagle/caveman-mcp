package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type logEvent struct {
	header string
	frames []string
	lang   string
}

var (
	goGoroutineRe = regexp.MustCompile(`(?m)^goroutine \d+ \[`)
	goFrameRe     = regexp.MustCompile(`^\t(.+\.go:\d+)`)
	goFuncRe      = regexp.MustCompile(`^(\S+)\(`)
	pyTracebackRe = regexp.MustCompile(`(?m)^Traceback \(most recent call last\):`)
	pyFrameRe     = regexp.MustCompile(`^\s+File "(.+)", line (\d+), in (.+)`)
	pyExceptionRe = regexp.MustCompile(`^(\w+(?:\.\w+)*(?:Error|Exception|Warning)|KeyError|ValueError|TypeError|RuntimeError|AttributeError|ImportError|NameError|IndexError|OSError|IOError|FileNotFoundError|PermissionError|StopIteration):? `)
	jsErrorRe     = regexp.MustCompile(`(?m)^(?:\w+)?Error:`)
	jsFrameRe     = regexp.MustCompile(`^\s+at (\S+) \((.+:\d+:\d+)\)`)
	javaExRe      = regexp.MustCompile(`(?m)^Exception in thread|(?m)^\w[\w.]+Exception:`)
	javaFrameRe   = regexp.MustCompile(`^\s+at ([\w.$]+)\((.+\.java:\d+)\)`)
	rustPanicRe   = regexp.MustCompile(`(?m)^thread '.+' panicked at`)

	stdlibPatterns = []*regexp.Regexp{
		regexp.MustCompile(`/usr/local/go/src/`),
		regexp.MustCompile(`/usr/lib/python`),
		regexp.MustCompile(`^node:internal/`),
		regexp.MustCompile(`node_modules/`),
		regexp.MustCompile(`\bjava\.`),
		regexp.MustCompile(`\bsun\.`),
		regexp.MustCompile(`\bcom\.sun\.`),
		regexp.MustCompile(`^net/http\.`),
		regexp.MustCompile(`^runtime\.`),
		regexp.MustCompile(`^reflect\.`),
		regexp.MustCompile(`^testing\.`),
	}
)

func isStdlibFrame(frame string) bool {
	for _, re := range stdlibPatterns {
		if re.MatchString(frame) {
			return true
		}
	}
	return false
}

func detectLang(text string) string {
	switch {
	case goGoroutineRe.MatchString(text):
		return "go"
	case pyTracebackRe.MatchString(text):
		return "python"
	case jsErrorRe.MatchString(text):
		return "javascript"
	case javaExRe.MatchString(text):
		return "java"
	case rustPanicRe.MatchString(text):
		return "rust"
	default:
		return "generic"
	}
}

func ParseLog(text string) string {
	lang := detectLang(text)
	events := splitEvents(text, lang)
	if len(events) == 0 {
		return text
	}

	counts := map[string]int{}
	var order []string
	for _, e := range events {
		key := e.header + "|" + strings.Join(e.frames, "|")
		if counts[key] == 0 {
			order = append(order, key)
		}
		counts[key]++
	}

	sort.Slice(order, func(i, j int) bool {
		return counts[order[i]] > counts[order[j]]
	})

	eventMap := map[string]*logEvent{}
	for _, e := range events {
		key := e.header + "|" + strings.Join(e.frames, "|")
		if eventMap[key] == nil {
			eventMap[key] = e
		}
	}

	var sb strings.Builder
	for _, key := range order {
		e := eventMap[key]
		count := counts[key]
		sb.WriteString(e.header)
		if count > 1 {
			sb.WriteString(fmt.Sprintf(" ×%d", count))
		}
		sb.WriteRune('\n')
		for _, f := range e.frames {
			sb.WriteString("  " + f + "\n")
		}
		sb.WriteRune('\n')
	}
	return strings.TrimSpace(sb.String())
}

func splitEvents(text, lang string) []*logEvent {
	var events []*logEvent
	var current *logEvent
	var currentFunc string

	flush := func() {
		if current != nil && current.header != "" {
			events = append(events, current)
			current = nil
		}
	}

	for _, line := range strings.Split(text, "\n") {
		switch lang {
		case "go":
			if goGoroutineRe.MatchString(line) || strings.HasPrefix(line, "panic:") ||
				strings.HasPrefix(line, "ERROR ") || strings.HasPrefix(line, "FATAL ") {
				flush()
				current = &logEvent{header: strings.TrimSpace(line), lang: "go"}
				currentFunc = ""
			} else if current != nil {
				if m := goFuncRe.FindStringSubmatch(line); m != nil && !isStdlibFrame(m[1]) {
					currentFunc = m[1]
				} else if m := goFrameRe.FindStringSubmatch(line); m != nil && currentFunc != "" && !isStdlibFrame(line) {
					current.frames = append(current.frames, currentFunc+" "+m[1])
					currentFunc = ""
				}
			}

		case "python":
			if pyTracebackRe.MatchString(line) {
				flush()
				current = &logEvent{header: "Traceback", lang: "python"}
			} else if pyExceptionRe.MatchString(line) {
				if current != nil {
					current.header = strings.TrimSpace(line)
				} else {
					flush()
					current = &logEvent{header: strings.TrimSpace(line), lang: "python"}
				}
			} else if current != nil {
				if m := pyFrameRe.FindStringSubmatch(line); m != nil && !isStdlibFrame(m[1]) {
					current.frames = append(current.frames, m[3]+" "+m[1]+":"+m[2])
				}
			}

		case "javascript":
			if jsErrorRe.MatchString(line) {
				flush()
				current = &logEvent{header: strings.TrimSpace(line), lang: "javascript"}
			} else if current != nil {
				if m := jsFrameRe.FindStringSubmatch(line); m != nil && !isStdlibFrame(m[1]) {
					current.frames = append(current.frames, m[1]+" "+m[2])
				}
			}

		case "java":
			if javaExRe.MatchString(line) {
				flush()
				current = &logEvent{header: strings.TrimSpace(line), lang: "java"}
			} else if current != nil {
				if m := javaFrameRe.FindStringSubmatch(line); m != nil && !isStdlibFrame(m[1]) {
					current.frames = append(current.frames, m[1]+"("+m[2]+")")
				}
			}

		default:
			if rustPanicRe.MatchString(line) || strings.HasPrefix(line, "ERROR") ||
				strings.HasPrefix(line, "FATAL") || strings.HasPrefix(line, "PANIC") {
				flush()
				current = &logEvent{header: strings.TrimSpace(line), lang: "generic"}
			}
		}
	}
	flush()
	return events
}
