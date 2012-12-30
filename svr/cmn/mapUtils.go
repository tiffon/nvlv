package cmn

import (
	"fmt"
)

// Add to data set of key-value pairs, passed in as strings where 
// strings are in format key1, value1, key2, value2, etc. Returns
// the set of keys added.
func AppendKVPs(data map[string]interface{}, keyValPairs []interface{}) (keysAdded []string, err error) {
	if len(keyValPairs) == 0 {
		return nil, nil
	}
	if len(keyValPairs)%2 != 0 {
		return nil, fmt.Errorf("invalid number of arguments, should be divisible by 2: %d", len(keyValPairs))
	}

	keysAdded = make([]string, 0, int(len(keyValPairs)/2))
	key := ""
	for _, elm := range keyValPairs {
		if len(key) > 0 {
			data[key] = elm
			keysAdded = append(keysAdded, key)
			key = ""
		} else {
			key = fmt.Sprintf("%v", elm)
		}
	}
	return keysAdded, nil
}
