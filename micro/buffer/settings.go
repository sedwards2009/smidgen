package buffer

import (
	"crypto/md5"
	"reflect"

	"github.com/sedwards2009/smidgen/micro/config"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
)

func (b *Buffer) DoSetOptionNative(option string, nativeValue any) {
	oldValue := b.Settings[option]
	if reflect.DeepEqual(oldValue, nativeValue) {
		return
	}

	b.Settings[option] = nativeValue

	if option == "fastdirty" {
		if !nativeValue.(bool) {
			if b.Size() > LargeFileThreshold {
				b.Settings["fastdirty"] = true
			} else {
				if !b.isModified {
					b.calcHash(&b.origHash)
				} else {
					// prevent using an old stale origHash value
					b.origHash = [md5.Size]byte{}
				}
			}
		}
	} else if option == "fileformat" {
		switch b.Settings["fileformat"].(string) {
		case "unix":
			b.Endings = FFUnix
		case "dos":
			b.Endings = FFDos
		}
		b.setModified()
	} else if option == "syntax" {
		if !nativeValue.(bool) {
			b.ClearMatches()
		} else {
			b.UpdateRules()
		}
	} else if option == "encoding" {
		enc, err := htmlindex.Get(b.Settings["encoding"].(string))
		if err != nil {
			enc = unicode.UTF8
			b.Settings["encoding"] = "utf-8"
		}
		b.encoding = enc
		b.setModified()
	}
}

func (b *Buffer) SetOptionNative(option string, nativeValue any) error {
	if err := config.OptionIsValid(option, nativeValue); err != nil {
		return err
	}

	b.DoSetOptionNative(option, nativeValue)
	return nil
}

// SetOption sets a given option to a value just for this buffer
// func (b *Buffer) SetOption(option, value string) error {
// 	if _, ok := b.Settings[option]; !ok {
// 		return config.ErrInvalidOption
// 	}

// 	nativeValue, err := config.GetNativeValue(option, value)
// 	if err != nil {
// 		return err
// 	}

// 	return b.SetOptionNative(option, nativeValue)
// }
