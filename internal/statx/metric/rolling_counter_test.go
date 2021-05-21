package metric

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRollingCounter_Add(t *testing.T) {
	size := 3
	bucketDuration := time.Second

	r := NewRollingCounter(
		WithRollingCounterOptsSize(size),
		WithRollingCounterOptsBucketDuration(bucketDuration),
	)

	listBuckets := func() [][]float64 {
		buckets := make([][]float64, 0)
		r.Reduce(func(i Iterator) float64 {
			for i.Next() {
				bucket := i.Bucket()
				buckets = append(buckets, bucket.Points)
			}
			return 0.0
		})
		return buckets
	}

	assert.Equal(t, [][]float64{{}, {}, {}}, listBuckets())
	r.Add(1)
	assert.Equal(t, [][]float64{{}, {}, {1}}, listBuckets())

	time.Sleep(time.Second)
	r.Add(2)
	r.Add(3)
	assert.Equal(t, [][]float64{{}, {1}, {5}}, listBuckets())

	time.Sleep(time.Second)
	r.Add(4)
	r.Add(5)
	r.Add(6)
	assert.Equal(t, [][]float64{{1}, {5}, {15}}, listBuckets())

	time.Sleep(time.Second)
	r.Add(7)
	assert.Equal(t, [][]float64{{5}, {15}, {7}}, listBuckets())
}
