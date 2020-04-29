package filewatcher

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
)

func TestFileWatcher(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())

	writeCount := 0
	removed := false
	fw, err := NewFileWatcher(func(event fsnotify.Event) {
		if event.Op&fsnotify.Write == fsnotify.Write {
			writeCount++
		} else if event.Op&fsnotify.Remove == fsnotify.Remove {
			removed = true
		}
	}, func(err error) {
		t.Fatal(err)
	})
	assert.NoError(t, err)

	go fw.Start(make(chan struct{}))
	_, err = tempFile.Write([]byte("test0"))
	assert.NoError(t, err)
	// Sleep for 1 second to allow event propagation
	time.Sleep(1 * time.Second)
	assert.Equal(t, 0, writeCount)
	assert.Equal(t, false, removed)

	err = fw.Add(tempFile.Name())
	assert.NoError(t, err)

	_, err = tempFile.Write([]byte("test1"))
	assert.NoError(t, err)
	// Sleep for 1 second to allow event propagation
	time.Sleep(1 * time.Second)
	assert.Equal(t, 1, writeCount)
	assert.Equal(t, false, removed)

	_, err = tempFile.Write([]byte("test2"))
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 2, writeCount)
	assert.Equal(t, false, removed)

	err = os.Remove(tempFile.Name())
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 2, writeCount)
	assert.Equal(t, true, removed)
}
