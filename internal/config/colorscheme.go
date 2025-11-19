package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Colorscheme is a map from string to style -- it represents a colorscheme
type Colorscheme map[string]tcell.Style

// GetColor takes in a syntax group and returns the colorscheme's style for that group
func (colorscheme Colorscheme) GetColor(color string) tcell.Style {
	var st tcell.Style
	if color == "" {
		return colorscheme.GetDefault()
	}

	groups := strings.Split(color, ".")
	if len(groups) > 1 {
		curGroup := ""
		for i, g := range groups {
			if i != 0 {
				curGroup += "."
			}
			curGroup += g
			if style, ok := colorscheme[curGroup]; ok {
				st = style
			}
		}
	} else if style, ok := colorscheme[color]; ok {
		st = style
	} else {
		st = StringToStyle(color, colorscheme.GetDefault())
	}

	return st
}

func (colorscheme Colorscheme) GetDefault() tcell.Style {
	return colorscheme["default"]
}

// ColorschemeExists checks if a given colorscheme exists
// func ColorschemeExists(colorschemeName string) bool {
// 	return FindRuntimeFile(RTColorscheme, colorschemeName) != nil
// }

// InitColorscheme picks and initializes the colorscheme when micro starts

// LoadDefaultColorscheme loads the default colorscheme from $(ConfigDir)/colorschemes
// func LoadDefaultColorscheme() (map[string]tcell.Style, error) {
// 	var parsedColorschemes []string
// 	return LoadColorscheme(GlobalSettings["colorscheme"].(string), &parsedColorschemes)
// }

// LoadColorscheme loads the given colorscheme from a directory
// func LoadColorscheme(colorschemeName string, parsedColorschemes *[]string) (map[string]tcell.Style, error) {
// 	c := make(map[string]tcell.Style)
// 	file := FindRuntimeFile(RTColorscheme, colorschemeName)
// 	if file == nil {
// 		return c, errors.New(colorschemeName + " is not a valid colorscheme")
// 	}
// 	if data, err := file.Data(); err != nil {
// 		return c, errors.New("Error loading colorscheme: " + err.Error())
// 	} else {
// 		var err error
// 		c, err = ParseColorscheme(file.Name(), string(data), parsedColorschemes)
// 		if err != nil {
// 			return c, err
// 		}
// 	}
// 	return c, nil
// }

// ParseColorscheme parses the text definition for a colorscheme and returns the corresponding object
// Colorschemes are made up of color-link statements linking a color group to a list of colors
// For example, color-link keyword (blue,red) makes all keywords have a blue foreground and
// red background
func ParseColorscheme(text string) Colorscheme {
	lines := strings.Split(text, "\n")

	c := make(Colorscheme)

	cleanLines := []string{}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" ||
			strings.TrimSpace(line)[0] == '#' {
			// Ignore this line
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	defaultStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	c["default"] = defaultStyle
	for _, line := range cleanLines {
		link, style := parseColorLine(line, defaultStyle)
		if link == "default" {
			defaultStyle = style
			break
		}
	}

	for _, line := range cleanLines {
		link, style := parseColorLine(line, defaultStyle)
		if link != "" {
			c[link] = style
		}
	}

	return c
}

func parseColorLine(line string, defaultStyle tcell.Style) (string, tcell.Style) {
	parser := regexp.MustCompile(`color-link\s+(\S*)\s+"(.*)"`)
	matches := parser.FindSubmatch([]byte(line))
	if len(matches) == 3 {
		link := string(matches[1])
		colors := string(matches[2])

		style := StringToStyle(colors, defaultStyle)
		return link, style
	} else {
		fmt.Println("Color-link statement is not valid:", line)
	}
	return "", defaultStyle
}

// StringToStyle returns a style from a string
// The strings must be in the format "extra foregroundcolor,backgroundcolor"
// The 'extra' can be bold, reverse, or underline
func StringToStyle(str string, defaultStyle tcell.Style) tcell.Style {
	var fg, bg string
	spaceSplit := strings.Split(str, " ")
	var split []string
	if len(spaceSplit) > 1 {
		split = strings.Split(spaceSplit[1], ",")
	} else {
		split = strings.Split(str, ",")
	}
	if len(split) > 1 {
		fg, bg = split[0], split[1]
	} else {
		fg = split[0]
	}
	fg = strings.TrimSpace(fg)
	bg = strings.TrimSpace(bg)

	var fgColor, bgColor tcell.Color
	var ok bool
	if fg == "" {
		fgColor, _, _ = defaultStyle.Decompose()
	} else {
		fgColor, ok = StringToColor(fg)
		if !ok {
			fgColor, _, _ = defaultStyle.Decompose()
		}

	}
	if bg == "" {
		_, bgColor, _ = defaultStyle.Decompose()
	} else {
		bgColor, ok = StringToColor(bg)
		if !ok {
			_, bgColor, _ = defaultStyle.Decompose()
		}
	}

	style := defaultStyle.Foreground(fgColor).Background(bgColor)
	if strings.Contains(str, "bold") {
		style = style.Bold(true)
	}
	if strings.Contains(str, "reverse") {
		style = style.Reverse(true)
	}
	if strings.Contains(str, "underline") {
		style = style.Underline(true)
	}
	return style
}

// StringToColor returns a tcell color from a string representation of a color
// We accept either bright... or light... to mean the brighter version of a color
func StringToColor(str string) (tcell.Color, bool) {
	switch str {
	case "black":
		return tcell.ColorBlack, true
	case "red":
		return tcell.ColorMaroon, true
	case "green":
		return tcell.ColorGreen, true
	case "yellow":
		return tcell.ColorOlive, true
	case "blue":
		return tcell.ColorNavy, true
	case "magenta":
		return tcell.ColorPurple, true
	case "cyan":
		return tcell.ColorTeal, true
	case "white":
		return tcell.ColorSilver, true
	case "brightblack", "lightblack":
		return tcell.ColorGray, true
	case "brightred", "lightred":
		return tcell.ColorRed, true
	case "brightgreen", "lightgreen":
		return tcell.ColorLime, true
	case "brightyellow", "lightyellow":
		return tcell.ColorYellow, true
	case "brightblue", "lightblue":
		return tcell.ColorBlue, true
	case "brightmagenta", "lightmagenta":
		return tcell.ColorFuchsia, true
	case "brightcyan", "lightcyan":
		return tcell.ColorAqua, true
	case "brightwhite", "lightwhite":
		return tcell.ColorWhite, true
	case "default":
		return tcell.ColorDefault, true
	default:
		// Check if this is a 256 color
		if num, err := strconv.Atoi(str); err == nil {
			return GetColor256(num), true
		}
		// Check if this is a truecolor hex value
		if len(str) == 7 && str[0] == '#' {
			return tcell.GetColor(str), true
		}
		return tcell.ColorDefault, false
	}
}

// GetColor256 returns the tcell color for a number between 0 and 255
func GetColor256(color int) tcell.Color {
	if color == 0 {
		return tcell.ColorDefault
	}
	return tcell.PaletteColor(color)
}
