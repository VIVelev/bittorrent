package io

import (
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
)

const perm fs.FileMode = 0755

func (tf *TorrentFile) WriteToFile(data []byte) error {
	if tf.IsMultiFile {
		return writeMultiFile(tf.Name, tf.Files, data)
	}

	return writeSingleFile(tf.Name, data)
}

func writeSingleFile(name string, data []byte) error {
	return ioutil.WriteFile(name, data, perm)
}

func writeMultiFile(dirname string, files []bencodeFile, data []byte) error {
	if err := os.Mkdir(dirname, perm); err != nil {
		return err
	}
	if err := os.Chdir(dirname); err != nil {
		return err
	}

	begin := 0
	for _, f := range files {
		buf := make([]byte, f.Length)
		copy(buf, data[begin:begin+f.Length])
		begin += f.Length

		l := len(f.Path) - 1
		fileName := f.Path[l]
		pathToFile := strings.Join(f.Path[:l], "/")
		if _, err := os.Stat(pathToFile + fileName); os.IsNotExist(err) {
			os.MkdirAll(pathToFile, perm)
		}
		if err := ioutil.WriteFile(pathToFile+fileName, buf, perm); err != nil {
			return err
		}
	}

	return nil
}
