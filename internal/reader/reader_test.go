package reader

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCSVReader_Read_FileNotExist(t *testing.T) {
	logger := zap.NewNop()
	r := NewCSVReader("not_exists.csv", ';', -1, logger)

	ctx := t.Context()
	_, errCh := r.Read(ctx)

	select {
	case err := <-errCh:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access path error")
	case <-time.After(time.Second):
		t.Fatal("expected error, but it timed out")
	}
}

func TestCSVReader_Read_SingleFile_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test.csv")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := "col1;col2\n1;2\n3;4\n"
	_, err = tmpFile.Write([]byte(content))
	assert.NoError(t, err)
	tmpFile.Close()

	logger := zap.NewNop()
	r := NewCSVReader(tmpFile.Name(), ';', -1, logger)

	ctx := t.Context()
	recCh, errCh := r.Read(ctx)

	select {
	case recs := <-recCh:
		assert.Equal(t, 3, len(recs))
		assert.Equal(t, []string{"col1", "col2"}, recs[0])
	case err := <-errCh:
		t.Fatalf("not expect error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timeout reading records...")
	}
}

func TestCSVReader_Read_Directory_Success(t *testing.T) {
	tmpDir := t.TempDir()

	csvDir := filepath.Join(tmpDir, "csvs")
	err := os.Mkdir(csvDir, 0755)
	assert.NoError(t, err)

	file1 := filepath.Join(csvDir, "f1.csv")
	err = os.WriteFile(file1, []byte("a;b\n1;2\n"), 0600)
	assert.NoError(t, err)

	file2 := filepath.Join(csvDir, "f2.csv")
	err = os.WriteFile(file2, []byte("x;y\n3;4\n"), 0600)
	assert.NoError(t, err)

	logger := zap.NewNop()
	r := NewCSVReader(csvDir, ';', -1, logger)

	ctx := t.Context()
	recCh, errCh := r.Read(ctx)

	got := 0
loop:
	for {
		select {
		case recs, ok := <-recCh:
			if !ok {
				break loop
			}
			assert.True(t, len(recs) >= 2)
			got++
		case err, ok := <-errCh:
			if ok {
				t.Logf("ignored error in walk: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
	assert.Equal(t, 2, got, "expected exactly 2 files read")
}

func TestCSVReader_readFile_Error(t *testing.T) {
	logger := zap.NewNop()
	r := NewCSVReader("fake.csv", ';', 2, logger)

	tmpFile, err := os.CreateTemp(t.TempDir(), "bad.csv")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte("a;b;c\n1;2\n"))
	assert.NoError(t, err)
	tmpFile.Close()

	_, err = r.readFile(tmpFile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file csv read error")
}
