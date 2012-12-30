package svr

import (
	"fmt"
	"os"
	"time"
)

func getSsnSpace() (ssnDir, ssnLogFile string, err error) {

	sep := string(os.PathSeparator)
	t := time.Now()
	ssnDir = ssnBaseDir
	err = os.Mkdir(ssnDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return
	}

	ssnDir += sep + t.Format("2006-01-02T15_04_05Z07_00")

	// fine for now, if need better switch to GUIDs
	nm := ssnDir + "__0"

	err = os.Mkdir(nm, os.ModePerm)
	if os.IsExist(err) {
		i := 1
		// var nmi string
		for ; os.IsExist(err) && i < 10; i++ {
			nm = fmt.Sprint(ssnDir, "__", i)
			err = os.Mkdir(nm, os.ModePerm)
		}
		if err != nil {
			return
		}
	} else if err != nil {
		return
	}
	ssnDir = nm

	ssnLogFile = ssnDir + sep + "programOut.log"
	_, err = os.Create(ssnLogFile)
	if err != nil {
		return
	}

	return ssnDir, ssnLogFile, nil
}
