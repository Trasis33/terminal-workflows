package highlight

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
)

var templateParamPattern = regexp.MustCompile(`\{\{(\w+)(?::([^}]*))?\}\}`)

// TokenStyles maps Chroma token types to lipgloss styles.
type TokenStyles map[chroma.TokenType]lipgloss.Style

// TokenStylesFromColors creates token styles from ANSI-256 color codes.
func TokenStylesFromColors(primary, secondary, tertiary, dim, text string) TokenStyles {
	primaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(primary))
	secondaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(secondary))
	tertiaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tertiary))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(dim))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(text))

	return TokenStyles{
		chroma.Keyword:               primaryStyle,
		chroma.KeywordReserved:       primaryStyle,
		chroma.NameBuiltin:           secondaryStyle,
		chroma.LiteralString:         tertiaryStyle,
		chroma.LiteralStringDouble:   tertiaryStyle,
		chroma.LiteralStringSingle:   tertiaryStyle,
		chroma.LiteralStringBacktick: tertiaryStyle,
		chroma.Comment:               dimStyle,
		chroma.CommentSingle:         dimStyle,
		chroma.NameVariable:          secondaryStyle,
		chroma.Operator:              textStyle,
		chroma.Punctuation:           textStyle,
		chroma.LiteralNumber:         tertiaryStyle,
		chroma.GenericError:          textStyle,
	}
}

// Shell highlights a shell command string using Chroma bash lexer and lipgloss styling.
// Returns original input when highlighting cannot be applied.
func Shell(command string, styles TokenStyles) string {
	if command == "" || len(styles) == 0 {
		return command
	}

	lexer := lexers.Get("bash")
	if lexer == nil {
		return command
	}

	lexer = chroma.Coalesce(lexer)
	preprocessed, sentinels := replaceTemplateParams(command)
	iterator, err := lexer.Tokenise(nil, preprocessed)
	if err != nil {
		return command
	}

	paramStyle := templateParamStyle(styles)
	var out strings.Builder

	for token := iterator(); token != chroma.EOF; token = iterator() {
		value := token.Value
		if value == "" {
			continue
		}
		rendered := renderTokenValue(value, sentinels, styleForToken(token.Type, styles), paramStyle)
		out.WriteString(rendered)
	}

	return out.String()
}

// ShellPlain returns the plain text representation used for width calculations.
func ShellPlain(command string) string {
	return command
}

func replaceTemplateParams(command string) (string, map[string]string) {
	sentinels := make(map[string]string)
	idx := 0
	preprocessed := templateParamPattern.ReplaceAllStringFunc(command, func(match string) string {
		sentinel := "__WF_PARAM_" + strconv.Itoa(idx) + "__"
		sentinels[sentinel] = match
		idx++
		return sentinel
	})
	return preprocessed, sentinels
}

func renderTokenValue(value string, sentinels map[string]string, tokenStyle lipgloss.Style, paramStyle lipgloss.Style) string {
	if len(sentinels) == 0 {
		return tokenStyle.Render(value)
	}

	var out strings.Builder
	remaining := value
	for len(remaining) > 0 {
		nextIdx := -1
		nextSentinel := ""
		for sentinel := range sentinels {
			idx := strings.Index(remaining, sentinel)
			if idx >= 0 && (nextIdx == -1 || idx < nextIdx) {
				nextIdx = idx
				nextSentinel = sentinel
			}
		}

		if nextIdx == -1 {
			out.WriteString(tokenStyle.Render(remaining))
			break
		}

		if nextIdx > 0 {
			out.WriteString(tokenStyle.Render(remaining[:nextIdx]))
		}

		out.WriteString(paramStyle.Render(sentinels[nextSentinel]))
		remaining = remaining[nextIdx+len(nextSentinel):]
	}

	return out.String()
}

func templateParamStyle(styles TokenStyles) lipgloss.Style {
	if style, ok := styles[chroma.Keyword]; ok {
		return style.Bold(true)
	}
	if style, ok := styles[chroma.KeywordReserved]; ok {
		return style.Bold(true)
	}
	return lipgloss.NewStyle().Bold(true)
}

func styleForToken(tokenType chroma.TokenType, styles TokenStyles) lipgloss.Style {
	for current := tokenType; current != 0; current = current.Parent() {
		if style, ok := styles[current]; ok {
			return style
		}
	}
	return lipgloss.NewStyle()
}
