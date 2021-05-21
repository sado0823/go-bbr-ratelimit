package metric

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func initWindow(size int) *Window {
	window := NewWindow(WithWindowSize(size))

	for i := 0; i < size; i++ {
		window.AppendBucketPoint(i, 1.0)
	}
	return window
}

func TestWindow_Reset(t *testing.T) {

	size := 3
	window := initWindow(size)
	window.Reset()

	for i := 0; i < size; i++ {
		assert.Equal(t, 0, len(window.Bucket(i).Points))
	}

}

func TestWindow_ResetBucket(t *testing.T) {

	size := 3
	window := initWindow(size)
	window.ResetBucket(1)

	assert.Equal(t, 0, len(window.Bucket(1).Points))

	assert.Equal(t, 1, len(window.Bucket(0).Points))
	assert.Equal(t, 1, len(window.Bucket(2).Points))
}

func TestWindow_ResetBuckets(t *testing.T) {

	size := 3
	window := initWindow(size)

	window.ResetBuckets([]int{1, 2})
	assert.Equal(t, 0, len(window.Bucket(1).Points))
	assert.Equal(t, 0, len(window.Bucket(2).Points))
	assert.Equal(t, 1, len(window.Bucket(0).Points))

}

func TestWindow_AppendBucketPoint(t *testing.T) {

	size := 3
	window := initWindow(size)

	window.AppendBucketPoint(1, 666)

	assert.Equal(t, 2, len(window.Bucket(1).Points))
	assert.EqualValues(t, 2, window.Bucket(1).Count)
	assert.EqualValues(t, 666, window.Bucket(1).Points[1])

	assert.Equal(t, 1, len(window.Bucket(2).Points))
	assert.Equal(t, 1, len(window.Bucket(0).Points))

}

func TestWindow_AddBucketPoint(t *testing.T) {
	size := 3
	window := initWindow(size)

	window.AddBucketPoint(0, 1)

	assert.EqualValues(t, 2, window.Bucket(0).Points[0])
}

func TestWindow_Size(t *testing.T) {

	size := 3
	window := initWindow(size)

	assert.Equal(t, 3, window.Size())
}
