package config

import (
	"errors"
	"runtime"
	"strings"

	"golang.org/x/text/encoding/htmlindex"
)

type optionValidator func(string, any) error

// a list of settings that need option validators
var optionValidators = map[string]optionValidator{
	"autosave":    validateNonNegativeValue,
	"clipboard":   validateChoice,
	"colorcolumn": validateNonNegativeValue,
	// "colorscheme":     validateColorscheme,
	"detectlimit":     validateNonNegativeValue,
	"encoding":        validateEncoding,
	"fileformat":      validateChoice,
	"helpsplit":       validateChoice,
	"matchbracestyle": validateChoice,
	"multiopen":       validateChoice,
	"pageoverlap":     validateNonNegativeValue,
	"scrollmargin":    validateNonNegativeValue,
	"scrollspeed":     validateNonNegativeValue,
	"tabsize":         validatePositiveValue,
}

// a list of settings with pre-defined choices
var OptionChoices = map[string][]string{
	"clipboard":       {"internal", "external", "terminal"},
	"fileformat":      {"unix", "dos"},
	"helpsplit":       {"hsplit", "vsplit"},
	"matchbracestyle": {"underline", "highlight"},
	"multiopen":       {"tab", "hsplit", "vsplit"},
}

// a list of settings that can be globally and locally modified and their
// default values
var defaultCommonSettings = map[string]any{
	"autoindent":      true,
	"colorcolumn":     float64(0),
	"cursorline":      true,
	"detectlimit":     float64(100),
	"diffgutter":      false,
	"encoding":        "utf-8",
	"eofnewline":      true,
	"fastdirty":       false,
	"fileformat":      defaultFileFormat(),
	"filetype":        "unknown",
	"hlsearch":        false,
	"hltaberrors":     false,
	"hltrailingws":    false,
	"indentchar":      " ", // Deprecated
	"keepautoindent":  false,
	"matchbrace":      true,
	"matchbraceleft":  true,
	"matchbracestyle": "underline",
	"pageoverlap":     float64(2),
	"relativeruler":   false,
	"rmtrailingws":    false,
	"ruler":           true,
	"scrollbar":       false,
	"scrollmargin":    float64(3),
	"scrollspeed":     float64(2),
	"showchars":       "",
	"smartpaste":      true,
	"softwrap":        false,
	"syntax":          true,
	"tabmovement":     false,
	"tabsize":         float64(4),
	"tabstospaces":    false,
	"useprimary":      true,
	"wordwrap":        false,
}

/*
// a list of settings that should only be globally modified and their
// default values

	var DefaultGlobalOnlySettings = map[string]any{
		"clipboard":     "external",
		"colorscheme":   "default",
		"divreverse":    true,
		"fakecursor":    false,
		"mouse":         true,
		"multiopen":     "tab",
		"parsecursor":   false,
		"paste":         false,
		"scrollbarchar": "|",
		"tabhighlight":  false,
		"tabreverse":    true,
	}

// a list of settings that should never be globally modified

	var LocalSettings = []string{
		"filetype",
		"readonly",
	}
*/
var (
	ErrInvalidOption    = errors.New("Invalid option")
	ErrInvalidValue     = errors.New("Invalid value")
	ErrOptNotToggleable = errors.New("Option not toggleable")

	// This is the raw parsed json
	parsedSettings     map[string]any
	settingsParseError bool
)

/*
// func writeFile(name string, txt []byte) error {
// 	return util.SafeWrite(name, txt, false)
// }
	func validateParsedSettings() error {
		var err error
		defaults := DefaultAllSettings()
		for k, v := range parsedSettings {
			if strings.HasPrefix(reflect.TypeOf(v).String(), "map") {
				if strings.HasPrefix(k, "ft:") {
					for k1, v1 := range v.(map[string]any) {
						if _, ok := defaults[k1]; ok {
							if e := verifySetting(k1, v1, defaults[k1]); e != nil {
								err = e
								parsedSettings[k].(map[string]any)[k1] = defaults[k1]
								continue
							}
						}
					}
				} else {
					if _, e := glob.Compile(k); e != nil {
						err = errors.New("Error with glob setting " + k + ": " + e.Error())
						delete(parsedSettings, k)
						continue
					}
					for k1, v1 := range v.(map[string]any) {
						if _, ok := defaults[k1]; ok {
							if e := verifySetting(k1, v1, defaults[k1]); e != nil {
								err = e
								parsedSettings[k].(map[string]any)[k1] = defaults[k1]
								continue
							}
						}
					}
				}
				continue
			}

			if k == "autosave" {
				// if autosave is a boolean convert it to float
				s, ok := v.(bool)
				if ok {
					if s {
						parsedSettings["autosave"] = 8.0
					} else {
						parsedSettings["autosave"] = 0.0
					}
				}
				continue
			}

			if _, ok := defaults[k]; ok {
				if e := verifySetting(k, v, defaults[k]); e != nil {
					err = e
					parsedSettings[k] = defaults[k]
					continue
				}
			}
		}
		return err
	}
*/
/*
func ParsedSettings() map[string]any {
	s := make(map[string]any)
	for k, v := range parsedSettings {
		s[k] = v
	}
	return s
}

func verifySetting(option string, value any, def any) error {
	valType := reflect.TypeOf(value)
	defType := reflect.TypeOf(def)
	assignable := defType.AssignableTo(valType)
	if !assignable {
		return fmt.Errorf("Error: setting '%s' has incorrect type (%s), using default value: %v (%s)", option, valType, def, defType)
	}

	if err := OptionIsValid(option, value); err != nil {
		return err
	}

	return nil
}

// UpdatePathGlobLocals scans the already parsed settings and sets the options locally
// based on whether the path matches a glob
// Must be called after ReadSettings
func UpdatePathGlobLocals(settings map[string]any, path string) {
	for k, v := range parsedSettings {
		if strings.HasPrefix(reflect.TypeOf(v).String(), "map") && !strings.HasPrefix(k, "ft:") {
			g, _ := glob.Compile(k)
			if g.MatchString(path) {
				for k1, v1 := range v.(map[string]any) {
					settings[k1] = v1
				}
			}
		}
	}
}

// UpdateFileTypeLocals scans the already parsed settings and sets the options locally
// based on whether the filetype matches to "ft:"
// Must be called after ReadSettings
func UpdateFileTypeLocals(settings map[string]any, filetype string) {
	for k, v := range parsedSettings {
		if strings.HasPrefix(reflect.TypeOf(v).String(), "map") && strings.HasPrefix(k, "ft:") {
			if filetype == k[3:] {
				for k1, v1 := range v.(map[string]any) {
					if k1 != "filetype" {
						settings[k1] = v1
					}
				}
			}
		}
	}
}
*/
func defaultFileFormat() string {
	if runtime.GOOS == "windows" {
		return "dos"
	}
	return "unix"
}

// DefaultCommonSettings returns a map of all common buffer settings
// and their default values
func DefaultCommonSettings() map[string]any {
	commonsettings := make(map[string]any)
	for k, v := range defaultCommonSettings {
		commonsettings[k] = v
	}
	return commonsettings
}

/*
// DefaultAllSettings returns a map of all common buffer & global-only settings
// and their default values
func DefaultAllSettings() map[string]any {
	allsettings := make(map[string]any)
	for k, v := range defaultCommonSettings {
		allsettings[k] = v
	}
	for k, v := range DefaultGlobalOnlySettings {
		allsettings[k] = v
	}
	return allsettings
}
*/
// OptionIsValid checks if a value is valid for a certain option
func OptionIsValid(option string, value any) error {
	if validator, ok := optionValidators[option]; ok {
		return validator(option, value)
	}

	return nil
}

// Option validators

func validatePositiveValue(option string, value any) error {
	nativeValue, ok := value.(float64)

	if !ok {
		return errors.New("Expected numeric type for " + option)
	}

	if nativeValue < 1 {
		return errors.New(option + " must be greater than 0")
	}

	return nil
}

func validateNonNegativeValue(option string, value any) error {
	nativeValue, ok := value.(float64)

	if !ok {
		return errors.New("Expected numeric type for " + option)
	}

	if nativeValue < 0 {
		return errors.New(option + " must be non-negative")
	}

	return nil
}

func validateChoice(option string, value any) error {
	if choices, ok := OptionChoices[option]; ok {
		val, ok := value.(string)
		if !ok {
			return errors.New("Expected string type for " + option)
		}

		for _, v := range choices {
			if val == v {
				return nil
			}
		}

		choicesStr := strings.Join(choices, ", ")
		return errors.New(option + " must be one of: " + choicesStr)
	}

	return errors.New("Option has no pre-defined choices")
}

// func validateColorscheme(option string, value any) error {
// 	colorscheme, ok := value.(string)

// 	if !ok {
// 		return errors.New("Expected string type for colorscheme")
// 	}

// 	if !ColorschemeExists(colorscheme) {
// 		return errors.New(colorscheme + " is not a valid colorscheme")
// 	}

// 	return nil
// }

func validateEncoding(option string, value any) error {
	_, err := htmlindex.Get(value.(string))
	return err
}
