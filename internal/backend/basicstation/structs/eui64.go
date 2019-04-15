package structs

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/brocaar/lorawan"
)

var euiRegexp = regexp.MustCompile(`\w{2}-\w{2}-\w{2}-\w{2}-\w{2}-\w{2}-\w{2}-\w{2}`)

// EUI64 implements the BasicStation EUI64 type.
type EUI64 lorawan.EUI64

// MarshalText encodes the EUI64 to a ID6 string.
func (e EUI64) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%x:%x:%x:%x", e[0:2], e[2:4], e[4:6], e[6:8])), nil
}

// UnmarshalText decodes the EUI64 from an ID6 or EUI string.
func (e *EUI64) UnmarshalText(text []byte) error {
	v := string(text)
	var eui lorawan.EUI64

	if euiRegexp.MatchString(v) {
		v = strings.Replace(v, "-", "", -1)
		if err := eui.UnmarshalText([]byte(v)); err != nil {
			return errors.Wrap(err, "unmarshal eui error")
		}
	} else {
		var blockI int
		blocks := strings.Split(v, ":")
		for i := 0; i < len(blocks); {
			if blocks[i] == "" {
				remaining := remainingBlocks(blocks[i:])
				i = len(blocks) - remaining
				blockI = 4 - remaining
			} else {
				v := "0000"[len(blocks[i]):] + blocks[i]
				b, err := hex.DecodeString(v)
				if err != nil {
					return errors.Wrap(err, "unmarshal eui block error")
				}
				for ii, bb := range b {
					eui[(blockI*2)+ii] = bb
				}

				blockI++
				i++
			}
		}
	}

	*e = EUI64(eui)
	return nil
}

func remainingBlocks(blocks []string) int {
	var i int
	for _, v := range blocks {
		if v != "" {
			i++
		}
	}
	return i
}
