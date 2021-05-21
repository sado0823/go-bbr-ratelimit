package metric

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func getRollingPolicy() *RollingPolicy {
	return NewRollingPolicy(
		NewWindow(WithWindowSize(10)),
		WithRollingPolicyOptsBucketDuration(300*time.Millisecond),
	)
}

func TestRollingPolicy_Add(t *testing.T) {

	policy := getRollingPolicy()

	time.Sleep(time.Millisecond * 400)
	policy.Add(1) // in window buckets offset 1


	time.Sleep(time.Millisecond * 201)
	policy.Add(1) // in window buckets offset 2

	assert.EqualValues(t, 1, policy.window.Bucket(1).Points[0])
	assert.EqualValues(t, 1, policy.window.Bucket(2).Points[0])

	// test func timespan return real span
	testCases := []map[string][]int{
		{
			"timeSleep":       []int{294, 3200},
			"offsetAndPoints": []int{0, 1, 0, 1},
		},
		{
			"timeSleep":       []int{305, 3200, 6400},
			"offsetAndPoints": []int{1, 1, 1, 1, 1, 1},
		},
	}
	for _, caseMap := range testCases {

		var totalTs int
		offsetAndPoints := caseMap["offsetAndPoints"]
		timeSleep := caseMap["timeSleep"]
		policyN := getRollingPolicy()
		for i, ts := range timeSleep {
			totalTs += ts
			time.Sleep(time.Duration(ts) * time.Millisecond)
			policyN.Add(1)
			offset, points := offsetAndPoints[2*i], offsetAndPoints[2*i+1]
			assert.EqualValues(t, points, policyN.window.Bucket(offset).Points[0])
		}
	}

}
