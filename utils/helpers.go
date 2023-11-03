package utils

import (
	"io"
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// BuffPool buffer pool
var BuffPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 64*1024)
	},
}

// LoadConfigFile from yaml file
func LoadConfigFile(configFilePath string, config interface{}) {
	configFile, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(configFile, config)
	if err != nil {
		log.Fatal(err)
	}
}

// CopyAndPrintError ditto
func CopyAndPrintError(dst io.Writer, src io.Reader, logger *Logger) int64 {
	buf := BuffPool.Get().([]byte)
	defer BuffPool.Put(buf)
	size, err := io.CopyBuffer(dst, src, buf)
	if err != nil && err != io.EOF {
		logger.Error("Error while copy %s", err)
	}
	return size
}
