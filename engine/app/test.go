package engine

import (
	"io/ioutil"
	"os"
	"strings"

	"kosmix.fr/streaming/kosmixutil"
)

func TestFilenames() {
	file, err := os.Open("files.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	for _, filename := range lines {
		if kosmixutil.IsVideoFile(filename) {
			println(filename, "is a video file", "title", kosmixutil.GetTitle(filename))
		}
	}
}
