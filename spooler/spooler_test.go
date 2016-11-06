package spooler

import (
	"regexp"
	"testing"
)

func TestImagePattern(t *testing.T) {
	testData := map[string]bool{
		"name.JPG":  true,
		"name.JPEG": true,
		"name.PNG":  true,
		"name.jpg":  true,
		"name.jpeg": true,
		"name.png":  true,
		"name.foo":  false,
		"name.mov":  false,
		"name.bmp":  false,
	}

	for filename, expected := range testData {
		got, err := regexp.MatchString(ImagePattern, filename)
		if err != nil {
			t.Errorf("Unexpected error matching filename '%s' against pattern /%s/: %s", filename, ImagePattern, err)
		}
		if expected != got {
			t.Errorf("'%s' should match pattern /%s/", filename, ImagePattern)
		}
	}
}
