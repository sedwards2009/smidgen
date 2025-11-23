package action

import (
	"errors"
	"log"
	"regexp"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

func BindingMappingToKeyTree(bufdefaults map[string]string) *KeyTree {
	keyTree := NewKeyTree()
	for k, v := range bufdefaults {
		BindKey(keyTree, k, v)
	}
	return keyTree
}

func BindKey(keyTree *KeyTree, k, v string) {
	event, err := findEvent(k)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	BufMapEvent(keyTree, event, v)
}

var r = regexp.MustCompile("<(.+?)>")

func findEvents(k string) (b KeySequenceEvent, ok bool, err error) {
	var events []Event = nil
	for len(k) > 0 {
		groups := r.FindStringSubmatchIndex(k)

		if len(groups) > 3 {
			if events == nil {
				events = make([]Event, 0, 3)
			}

			e, ok := findSingleEvent(k[groups[2]:groups[3]])
			if !ok {
				return KeySequenceEvent{}, false, errors.New("Invalid event " + k[groups[2]:groups[3]])
			}

			events = append(events, e)

			k = k[groups[3]+1:]
		} else {
			return KeySequenceEvent{}, false, nil
		}
	}

	return KeySequenceEvent{events}, true, nil
}

type KeyDesc struct {
	KeyCode   tcell.Key
	Modifiers tcell.ModMask
	R         rune
}

func ParseKeyboardSequence(k string) (KeyDesc, bool) {
	ev, ok := findSingleEvent(k)
	if !ok {
		return KeyDesc{}, false
	}
	if ke, ok := ev.(KeyEvent); ok {
		return KeyDesc{
			KeyCode:   ke.code,
			Modifiers: ke.mod,
			R:         ke.r,
		}, true
	}
	return KeyDesc{}, false
}

// findSingleEvent will find binding Key 'b' using string 'k'
func findSingleEvent(k string) (b Event, ok bool) {
	modifiers := tcell.ModNone

	// First, we'll strip off all the modifiers in the name and add them to the
	// ModMask
modSearch:
	for {
		switch {
		case strings.HasPrefix(k, "-") && k != "-":
			// We optionally support dashes between modifiers
			k = k[1:]
		case strings.HasPrefix(k, "Ctrl") && k != "CtrlH":
			// CtrlH technically does not have a 'Ctrl' modifier because it is really backspace
			k = k[4:]
			modifiers |= tcell.ModCtrl
		case strings.HasPrefix(k, "Alt"):
			k = k[3:]
			modifiers |= tcell.ModAlt
		case strings.HasPrefix(k, "Shift"):
			k = k[5:]
			modifiers |= tcell.ModShift
		case strings.HasPrefix(k, "\x1b"):
			return RawEvent{
				esc: k,
			}, true
		default:
			break modSearch
		}
	}

	if k == "" {
		return KeyEvent{}, false
	}

	// Control is handled in a special way, since the terminal sends explicitly
	// marked escape sequences for control keys
	// We should check for Control keys first
	if modifiers&tcell.ModCtrl != 0 {
		// see if the key is in bindingKeys with the Ctrl prefix.
		k = string(unicode.ToUpper(rune(k[0]))) + k[1:]
		if code, ok := keyEvents["Ctrl"+k]; ok {
			return KeyEvent{
				code: code,
				mod:  modifiers,
			}, true
		}
	}

	// See if we can find the key in bindingKeys
	if code, ok := keyEvents[k]; ok {
		return KeyEvent{
			code: code,
			mod:  modifiers,
		}, true
	}

	var mstate MouseState = MousePress
	if strings.HasSuffix(k, "Drag") {
		k = k[:len(k)-4]
		mstate = MouseDrag
	} else if strings.HasSuffix(k, "Release") {
		k = k[:len(k)-7]
		mstate = MouseRelease
	}
	// See if we can find the key in bindingMouse
	if code, ok := mouseEvents[k]; ok {
		return MouseEvent{
			btn:   code,
			mod:   modifiers,
			state: mstate,
		}, true
	}

	// If we were given one character, then we've got a rune.
	if len(k) == 1 {
		return KeyEvent{
			code: tcell.KeyRune,
			mod:  modifiers,
			r:    rune(k[0]),
		}, true
	}

	// We don't know what happened.
	return KeyEvent{}, false
}

func findEvent(k string) (Event, error) {
	var event Event
	event, ok, err := findEvents(k)
	if err != nil {
		return nil, err
	}

	if !ok {
		event, ok = findSingleEvent(k)
		if !ok {
			return nil, errors.New(k + " is not a bindable event")
		}
	}

	return event, nil
}

func eventsEqual(e1 Event, e2 Event) bool {
	seq1, ok1 := e1.(KeySequenceEvent)
	seq2, ok2 := e2.(KeySequenceEvent)
	if ok1 && ok2 {
		if len(seq1.keys) != len(seq2.keys) {
			return false
		}
		for i := 0; i < len(seq1.keys); i++ {
			if seq1.keys[i] != seq2.keys[i] {
				return false
			}
		}
		return true
	}

	return e1 == e2
}

var mouseEvents = map[string]tcell.ButtonMask{
	"MouseLeft":       tcell.ButtonPrimary,
	"MouseMiddle":     tcell.ButtonMiddle,
	"MouseRight":      tcell.ButtonSecondary,
	"MouseWheelUp":    tcell.WheelUp,
	"MouseWheelDown":  tcell.WheelDown,
	"MouseWheelLeft":  tcell.WheelLeft,
	"MouseWheelRight": tcell.WheelRight,
}

var keyEvents = map[string]tcell.Key{
	"Up":             tcell.KeyUp,
	"Down":           tcell.KeyDown,
	"Right":          tcell.KeyRight,
	"Left":           tcell.KeyLeft,
	"UpLeft":         tcell.KeyUpLeft,
	"UpRight":        tcell.KeyUpRight,
	"DownLeft":       tcell.KeyDownLeft,
	"DownRight":      tcell.KeyDownRight,
	"Center":         tcell.KeyCenter,
	"PageUp":         tcell.KeyPgUp,
	"PageDown":       tcell.KeyPgDn,
	"Home":           tcell.KeyHome,
	"End":            tcell.KeyEnd,
	"Insert":         tcell.KeyInsert,
	"Delete":         tcell.KeyDelete,
	"Help":           tcell.KeyHelp,
	"Exit":           tcell.KeyExit,
	"Clear":          tcell.KeyClear,
	"Cancel":         tcell.KeyCancel,
	"Print":          tcell.KeyPrint,
	"Pause":          tcell.KeyPause,
	"Backtab":        tcell.KeyBacktab,
	"F1":             tcell.KeyF1,
	"F2":             tcell.KeyF2,
	"F3":             tcell.KeyF3,
	"F4":             tcell.KeyF4,
	"F5":             tcell.KeyF5,
	"F6":             tcell.KeyF6,
	"F7":             tcell.KeyF7,
	"F8":             tcell.KeyF8,
	"F9":             tcell.KeyF9,
	"F10":            tcell.KeyF10,
	"F11":            tcell.KeyF11,
	"F12":            tcell.KeyF12,
	"F13":            tcell.KeyF13,
	"F14":            tcell.KeyF14,
	"F15":            tcell.KeyF15,
	"F16":            tcell.KeyF16,
	"F17":            tcell.KeyF17,
	"F18":            tcell.KeyF18,
	"F19":            tcell.KeyF19,
	"F20":            tcell.KeyF20,
	"F21":            tcell.KeyF21,
	"F22":            tcell.KeyF22,
	"F23":            tcell.KeyF23,
	"F24":            tcell.KeyF24,
	"F25":            tcell.KeyF25,
	"F26":            tcell.KeyF26,
	"F27":            tcell.KeyF27,
	"F28":            tcell.KeyF28,
	"F29":            tcell.KeyF29,
	"F30":            tcell.KeyF30,
	"F31":            tcell.KeyF31,
	"F32":            tcell.KeyF32,
	"F33":            tcell.KeyF33,
	"F34":            tcell.KeyF34,
	"F35":            tcell.KeyF35,
	"F36":            tcell.KeyF36,
	"F37":            tcell.KeyF37,
	"F38":            tcell.KeyF38,
	"F39":            tcell.KeyF39,
	"F40":            tcell.KeyF40,
	"F41":            tcell.KeyF41,
	"F42":            tcell.KeyF42,
	"F43":            tcell.KeyF43,
	"F44":            tcell.KeyF44,
	"F45":            tcell.KeyF45,
	"F46":            tcell.KeyF46,
	"F47":            tcell.KeyF47,
	"F48":            tcell.KeyF48,
	"F49":            tcell.KeyF49,
	"F50":            tcell.KeyF50,
	"F51":            tcell.KeyF51,
	"F52":            tcell.KeyF52,
	"F53":            tcell.KeyF53,
	"F54":            tcell.KeyF54,
	"F55":            tcell.KeyF55,
	"F56":            tcell.KeyF56,
	"F57":            tcell.KeyF57,
	"F58":            tcell.KeyF58,
	"F59":            tcell.KeyF59,
	"F60":            tcell.KeyF60,
	"F61":            tcell.KeyF61,
	"F62":            tcell.KeyF62,
	"F63":            tcell.KeyF63,
	"F64":            tcell.KeyF64,
	"CtrlSpace":      tcell.KeyCtrlSpace,
	"CtrlA":          tcell.KeyCtrlA,
	"CtrlB":          tcell.KeyCtrlB,
	"CtrlC":          tcell.KeyCtrlC,
	"CtrlD":          tcell.KeyCtrlD,
	"CtrlE":          tcell.KeyCtrlE,
	"CtrlF":          tcell.KeyCtrlF,
	"CtrlG":          tcell.KeyCtrlG,
	"CtrlH":          tcell.KeyCtrlH,
	"CtrlI":          tcell.KeyCtrlI,
	"CtrlJ":          tcell.KeyCtrlJ,
	"CtrlK":          tcell.KeyCtrlK,
	"CtrlL":          tcell.KeyCtrlL,
	"CtrlM":          tcell.KeyCtrlM,
	"CtrlN":          tcell.KeyCtrlN,
	"CtrlO":          tcell.KeyCtrlO,
	"CtrlP":          tcell.KeyCtrlP,
	"CtrlQ":          tcell.KeyCtrlQ,
	"CtrlR":          tcell.KeyCtrlR,
	"CtrlS":          tcell.KeyCtrlS,
	"CtrlT":          tcell.KeyCtrlT,
	"CtrlU":          tcell.KeyCtrlU,
	"CtrlV":          tcell.KeyCtrlV,
	"CtrlW":          tcell.KeyCtrlW,
	"CtrlX":          tcell.KeyCtrlX,
	"CtrlY":          tcell.KeyCtrlY,
	"CtrlZ":          tcell.KeyCtrlZ,
	"CtrlLeftSq":     tcell.KeyCtrlLeftSq,
	"CtrlBackslash":  tcell.KeyCtrlBackslash,
	"CtrlRightSq":    tcell.KeyCtrlRightSq,
	"CtrlCarat":      tcell.KeyCtrlCarat,
	"CtrlUnderscore": tcell.KeyCtrlUnderscore,
	"Tab":            tcell.KeyTab,
	"Esc":            tcell.KeyEsc,
	"Escape":         tcell.KeyEscape,
	"Enter":          tcell.KeyEnter,
	"Backspace":      tcell.KeyBackspace2,
	"OldBackspace":   tcell.KeyBackspace,

	// I renamed these keys to PageUp and PageDown but I don't want to break someone's keybindings
	"PgUp":   tcell.KeyPgUp,
	"PgDown": tcell.KeyPgDn,
}
