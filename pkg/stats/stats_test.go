package stats_test

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/stats"
)

func TestBasics(t *testing.T) {
	validateMap := make(map[stats.Type]time.Duration)
	stats.Init()
	stats.Put(stats.Total, time.Second*20)
	validateMap[stats.Total] = time.Second * 20

	stats.Update(stats.AssignedIDDel, time.Second*40)
	validateMap[stats.AssignedIDDel] = time.Second * 40

	stats.Update(stats.AssignedIDDel, time.Second*40)
	validateMap[stats.AssignedIDDel] = time.Second * 80

	stats.PrintSync()
	getAllStats := stats.GetAll()
	if reflect.DeepEqual(getAllStats, validateMap) != true {
		panic("Stats added did not come back")
	}
}

func TestConcurrency(t *testing.T) {
	stats.Init()
	var wg sync.WaitGroup
	var startWg sync.WaitGroup

	// Make sure all of them start at the same time so that there are more chances of sync issues.
	startWg.Add(1)
	count := 100
	duration := time.Second * 10
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			startWg.Wait()
			stats.Update(stats.AssignedIDList, duration)
		}(i)
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			startWg.Wait()
			stats.Get(stats.AssignedIDList)
		}(i)
	}

	// All of them should start together.
	startWg.Done()
	wg.Wait()

	if stats.Get(stats.AssignedIDList) != duration*time.Duration(count) {
		panic("Stats did not get incremented.")
	}
}
