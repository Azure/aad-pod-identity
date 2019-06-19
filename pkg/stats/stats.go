package stats

import (
	"time"

	"github.com/golang/glog"
)

var GlobalStats map[StatsType]time.Duration

type StatsType string

const (
	Total                StatsType = "Total"
	System               StatsType = "System"
	CurrentState         StatsType = "Gather current state"
	PodList              StatsType = "Pod listing"
	BindingList          StatsType = "Binding listing"
	IDList               StatsType = "ID listing"
	AssignedIDList       StatsType = "Assigned ID listing"
	CloudGet             StatsType = "Cloud provider get"
	CloudPut             StatsType = "Cloud provider put"
	K8sGet               StatsType = "K8s get"
	K8sPut               StatsType = "K8s put"
	FindAssignedIDDel    StatsType = "Find assigned ids to delete"
	FindAssignedIDCreate StatsType = "Find assigned ids to create"
	AssignedIDDel        StatsType = "Assigned ID deletion"
	AssignedIDAdd        StatsType = "Assigned ID addition"
	TotalIDDel           StatsType = "Total time to delete assigned IDs"
	TotalIDAdd           StatsType = "Total time to add assigned IDs"

	EventRecord StatsType = "Event recording"
)

func Init() {
	GlobalStats = make(map[StatsType]time.Duration)
}

func Put(key StatsType, val time.Duration) {
	if GlobalStats != nil {
		GlobalStats[key] = val
	}
}

func Get(key StatsType) time.Duration {
	if GlobalStats != nil {
		return GlobalStats[key]
	}
	return 0
}

func Update(key StatsType, val time.Duration) {
	if GlobalStats != nil {
		GlobalStats[key] = GlobalStats[key] + val
	}
}

func Print(key StatsType) {
	glog.Infof("%s: %s", key, GlobalStats[key])
}

func PrintSync() {
	glog.Infof("** Stats collected **")
	if GlobalStats != nil {
		//first we list the
		Print(PodList)
		Print(IDList)
		Print(BindingList)
		Print(AssignedIDList)
		Print(System)

		Print(CloudGet)
		Print(CloudPut)
		Print(AssignedIDAdd)
		Print(AssignedIDDel)

		Print(FindAssignedIDCreate)
		Print(FindAssignedIDDel)

		Print(TotalIDAdd)
		Print(TotalIDDel)

		Print(EventRecord)
		Print(Total)
	}
	glog.Infof("*********************")
}

func GetAll() map[StatsType]time.Duration {
	return GlobalStats
}

/*
//More sophisticated stats
type MICStat struct {
	msg        string
	timeTaken  time.Duration
	operations int64
}
*/
