package stats_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/Azure/aad-pod-identity/pkg/stats"
	"github.com/golang/glog"
)

func TestBasics(t *testing.T) {
	glog.Infof("Test started")
	validateMap := make(map[stats.StatsType]time.Duration, 0)
	stats.Init()
	stats.Put(stats.Total, time.Second*20)
	validateMap[stats.Total] = time.Second * 20

	stats.Put(stats.K8sPut, time.Second*30)
	validateMap[stats.K8sPut] = time.Second * 30

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
