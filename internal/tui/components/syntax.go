package components

import (
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/vesvai/vesvai/internal/tui/styles"
)

func langFromExt(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".hpp", ".cxx":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".ex", ".exs":
		return "elixir"
	case ".hs":
		return "haskell"
	case ".css", ".scss", ".less":
		return "css"
	case ".html", ".htm":
		return "html"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md":
		return "markdown"
	case ".sh", ".bash", ".zsh":
		return "bash"
	case ".sql":
		return "sql"
	case ".r", ".R":
		return "r"
	case ".lua":
		return "lua"
	case ".dart":
		return "dart"
	default:
		return ""
	}
}

func highlightLine(text string, filePath string, th styles.Theme, contentStyle tcell.Style) Line {
	lang := langFromExt(filePath)
	if lang == "" {
		return lineFromPlain(text, contentStyle)
	}
	return highlightByLang(text, lang, th, contentStyle)
}

func highlightByLangName(text string, lang string, th styles.Theme, contentStyle tcell.Style) Line {
	return highlightByLang(text, lang, th, contentStyle)
}

func lineFromPlain(text string, style tcell.Style) Line {
	var out Line
	for _, r := range text {
		out = append(out, Cell{R: r, S: style})
	}
	return out
}

func highlightByLang(text string, lang string, th styles.Theme, base tcell.Style) Line {
	tokens := tokenize(text, lang, th)
	var out Line
	for _, tok := range tokens {
		for _, r := range tok.Text {
			out = append(out, Cell{R: r, S: tok.Style})
		}
	}
	return out
}

type styledToken struct {
	Text  string
	Style tcell.Style
}

func tokenize(text string, lang string, th styles.Theme) []styledToken {
	keywords, types, builtins := langKeywords(lang)
	tokenStyle := th.Base().Foreground(th.Foreground).Background(th.Background)

	var tokens []styledToken
	i := 0
	for i < len(text) {
		if i+1 < len(text) && text[i] == '/' && text[i+1] == '/' {
			tokens = append(tokens, styledToken{Text: text[i:], Style: tokenStyle.Foreground(th.TokenComment)})
			break
		}
		if i+1 < len(text) && text[i] == '/' && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "*/")
			if end < 0 {
				tokens = append(tokens, styledToken{Text: text[i:], Style: tokenStyle.Foreground(th.TokenComment)})
				break
			}
			tokens = append(tokens, styledToken{Text: text[i : i+end+4], Style: tokenStyle.Foreground(th.TokenComment)})
			i += end + 4
			continue
		}
		if text[i] == '#' && (lang == "python" || lang == "bash" || lang == "ruby" || lang == "r" || lang == "elixir") {
			tokens = append(tokens, styledToken{Text: text[i:], Style: tokenStyle.Foreground(th.TokenComment)})
			break
		}

		if text[i] == '"' || text[i] == '\'' || text[i] == '`' {
			quote := text[i]
			j := i + 1
			for j < len(text) {
				if text[j] == '\\' {
					j += 2
					continue
				}
				if text[j] == quote {
					j++
					break
				}
				j++
			}
			tokens = append(tokens, styledToken{Text: text[i:j], Style: tokenStyle.Foreground(th.TokenString)})
			i = j
			continue
		}

		if text[i] >= '0' && text[i] <= '9' {
			j := i
			for j < len(text) && (text[j] >= '0' && text[j] <= '9' || text[j] == '.' || text[j] == 'x' || text[j] == 'X' || (text[j] >= 'a' && text[j] <= 'f') || (text[j] >= 'A' && text[j] <= 'F')) {
				j++
			}
			tokens = append(tokens, styledToken{Text: text[i:j], Style: tokenStyle.Foreground(th.TokenNumber)})
			i = j
			continue
		}

		if text[i] == '_' || (text[i] >= 'a' && text[i] <= 'z') || (text[i] >= 'A' && text[i] <= 'Z') {
			j := i
			for j < len(text) && (text[j] == '_' || (text[j] >= 'a' && text[j] <= 'z') || (text[j] >= 'A' && text[j] <= 'Z') || (text[j] >= '0' && text[j] <= '9')) {
				j++
			}
			word := text[i:j]
			if isKeyword(word, keywords) {
				tokens = append(tokens, styledToken{Text: word, Style: tokenStyle.Foreground(th.TokenKeyword)})
			} else if isKeyword(word, types) {
				tokens = append(tokens, styledToken{Text: word, Style: tokenStyle.Foreground(th.TokenType)})
			} else if isKeyword(word, builtins) {
				tokens = append(tokens, styledToken{Text: word, Style: tokenStyle.Foreground(th.TokenConstant)})
			} else if len(word) >= 2 && word[0] >= 'A' && word[0] <= 'Z' {
				tokens = append(tokens, styledToken{Text: word, Style: tokenStyle.Foreground(th.TokenFunction)})
			} else {
				tokens = append(tokens, styledToken{Text: word, Style: tokenStyle.Foreground(th.Foreground)})
			}
			i = j
			continue
		}

		tokens = append(tokens, styledToken{Text: string(text[i]), Style: tokenStyle.Foreground(th.TokenPunctuation)})
		i++
	}

	return tokens
}

func isKeyword(word string, set map[string]bool) bool {
	_, ok := set[word]
	return ok
}

func langKeywords(lang string) (keywords, types, builtins map[string]bool) {
	keywords = map[string]bool{}
	types = map[string]bool{}
	builtins = map[string]bool{}

	switch lang {
	case "go":
		keywords = map[string]bool{
			"break": true, "case": true, "chan": true, "const": true,
			"continue": true, "default": true, "defer": true, "else": true,
			"fallthrough": true, "for": true, "func": true, "go": true,
			"goto": true, "if": true, "import": true, "interface": true,
			"map": true, "package": true, "range": true, "return": true,
			"select": true, "struct": true, "switch": true, "type": true,
			"var": true,
		}
		types = map[string]bool{
			"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
			"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
			"float32": true, "float64": true,
			"bool": true, "string": true, "byte": true, "rune": true,
			"error": true, "any": true,
		}
		builtins = map[string]bool{
			"true": true, "false": true, "nil": true,
			"make": true, "new": true, "len": true, "cap": true, "append": true,
			"copy": true, "delete": true, "close": true, "panic": true, "recover": true,
			"print": true, "println": true,
		}

	case "typescript", "javascript":
		keywords = map[string]bool{
			"break": true, "case": true, "catch": true, "class": true,
			"const": true, "continue": true, "debugger": true, "default": true,
			"delete": true, "do": true, "else": true, "export": true,
			"extends": true, "finally": true, "for": true, "function": true,
			"if": true, "import": true, "in": true, "instanceof": true,
			"let": true, "new": true, "of": true, "return": true,
			"super": true, "switch": true, "this": true, "throw": true,
			"try": true, "typeof": true, "var": true, "void": true,
			"while": true, "with": true, "yield": true, "async": true,
			"await": true, "from": true, "as": true, "interface": true,
			"type": true, "enum": true, "implements": true,
		}
		types = map[string]bool{
			"number": true, "string": true, "boolean": true, "undefined": true,
			"null": true, "any": true, "void": true, "never": true,
			"object": true, "unknown": true,
		}
		builtins = map[string]bool{
			"true": true, "false": true, "undefined": true, "null": true,
			"console": true, "Math": true, "JSON": true, "Array": true,
			"Object": true, "String": true, "Number": true, "Boolean": true,
			"Promise": true, "Map": true, "Set": true, "Date": true,
			"Error": true, "RegExp": true,
		}

	case "python":
		keywords = map[string]bool{
			"False": true, "None": true, "True": true, "and": true, "as": true,
			"assert": true, "async": true, "await": true, "break": true,
			"class": true, "continue": true, "def": true, "del": true,
			"elif": true, "else": true, "except": true, "finally": true,
			"for": true, "from": true, "global": true, "if": true,
			"import": true, "in": true, "is": true, "lambda": true,
			"nonlocal": true, "not": true, "or": true, "pass": true,
			"raise": true, "return": true, "try": true, "while": true,
			"with": true, "yield": true,
		}
		types = map[string]bool{
			"int": true, "float": true, "bool": true, "str": true,
			"list": true, "dict": true, "tuple": true, "set": true,
			"bytes": true,
		}
		builtins = map[string]bool{
			"print": true, "len": true, "range": true, "type": true,
			"isinstance": true, "enumerate": true, "zip": true, "map": true,
			"filter": true, "sorted": true, "reversed": true, "open": true,
			"super": true, "self": true, "cls": true,
		}

	case "rust":
		keywords = map[string]bool{
			"as": true, "break": true, "const": true, "continue": true,
			"crate": true, "else": true, "enum": true, "extern": true,
			"false": true, "fn": true, "for": true, "if": true,
			"impl": true, "in": true, "let": true, "loop": true,
			"match": true, "mod": true, "move": true, "mut": true,
			"pub": true, "ref": true, "return": true, "self": true,
			"static": true, "struct": true, "super": true, "trait": true,
			"true": true, "type": true, "unsafe": true, "use": true,
			"where": true, "while": true, "async": true, "await": true,
			"dyn": true,
		}
		types = map[string]bool{
			"i8": true, "i16": true, "i32": true, "i64": true, "i128": true,
			"u8": true, "u16": true, "u32": true, "u64": true, "u128": true,
			"f32": true, "f64": true, "bool": true, "char": true,
			"String": true, "str": true, "Vec": true, "Option": true,
			"Result": true, "Box": true, "Rc": true, "Arc": true,
		}
		builtins = map[string]bool{
			"Some": true, "None": true, "Ok": true, "Err": true,
			"println": true, "format": true, "clone": true, "unwrap": true,
		}

	case "bash":
		keywords = map[string]bool{
			"if": true, "then": true, "else": true, "elif": true,
			"fi": true, "for": true, "while": true, "until": true,
			"do": true, "done": true, "case": true, "esac": true,
			"in": true, "function": true, "select": true,
			"break": true, "continue": true, "return": true, "exit": true,
			"local": true, "declare": true, "typeset": true, "readonly": true,
			"export": true, "unset": true, "shift": true,
			"time": true, "coproc": true,
		}
		types = map[string]bool{
			"-a": true, "-b": true, "-c": true, "-d": true, "-e": true,
			"-f": true, "-g": true, "-h": true, "-k": true, "-m": true,
			"-n": true, "-o": true, "-p": true, "-r": true, "-s": true,
			"-t": true, "-u": true, "-w": true, "-x": true, "-z": true,
		}
		builtins = map[string]bool{
			"echo": true, "printf": true, "read": true, "mapfile": true,
			"readarray": true, "cd": true, "pwd": true, "pushd": true,
			"popd": true, "dirs": true, "let": true, "eval": true,
			"exec": true, "trap": true, "wait": true, "bg": true,
			"fg": true, "jobs": true, "kill": true, "exit": true,
			"set": true, "shopt": true, "umask": true, "alias": true,
			"unalias": true, "hash": true, "type": true, "command": true,
			"builtin": true, "source": true, "test": true,
			"true": true, "false": true, "yes": true,
			"grep": true, "sed": true, "awk": true, "find": true,
			"sort": true, "uniq": true, "wc": true, "head": true,
			"tail": true, "cat": true, "less": true, "more": true,
			"ls": true, "cp": true, "mv": true, "rm": true, "mkdir": true,
			"chmod": true, "chown": true, "touch": true, "ln": true,
			"tar": true, "gzip": true, "gunzip": true, "curl": true,
			"wget": true, "ssh": true, "scp": true, "rsync": true,
			"git": true, "docker": true, "make": true, "go": true,
			"python": true, "node": true, "npm": true, "pip": true,
			"cargo": true, "rustc": true, "gcc": true, "g++": true,
		}
	}

	return keywords, types, builtins
}
