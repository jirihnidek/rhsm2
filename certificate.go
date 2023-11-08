package rhsm2

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

// writePemFile Tries to write content of PEM (cert or key) to file.
// If mode is not nil, then access permissions to give file will be modified
func writePemFile(filePath *string, pemFileContent *string, mode *os.FileMode) error {
	if len(*pemFileContent) == 0 {
		return fmt.Errorf("canceling writing pem file: %s, because provided content is empty", *filePath)
	}

	file, err := os.Create(*filePath)
	if err != nil {
		return err
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Error().Msgf("unable to close %s: %s", *filePath, err)
		}
	}(file)

	// Print content of cert using Fprint(), because
	// the string contains formatting sequences like \n
	_, err = fmt.Fprint(file, *pemFileContent)

	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	// Optionally try to change access permission to file
	if mode != nil {
		err = os.Chmod(*filePath, *mode)
		if err != nil {
			return fmt.Errorf("unable to change access permission of %s to (%v): %v", *filePath, *mode, err)
		}
	}

	log.Debug().Msgf("installed %s", *filePath)

	return nil
}
