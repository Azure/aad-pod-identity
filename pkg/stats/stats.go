package stats

import (
	"sync"
	"time"

	"k8s.io/klog"
)

var GlobalStats map[StatsType]time.Duration
var CountStats map[StatsType]int

var Mutex *sync.RWMutex

type StatsType string

const (
	Total                   StatsType = "Total"
	System                  StatsType = "System"
	CacheSync               StatsType = "CacheSync"
	CurrentState            StatsType = "Gather current state"
	PodList                 StatsType = "Pod listing"
	BindingList             StatsType = "Binding listing"
	IDList                  StatsType = "ID listing"
	ExceptionList           StatsType = "Pod Identity Exception listing"
	AssignedIDList          StatsType = "Assigned ID listing"
	CloudGet                StatsType = "Cloud provider get"
	CloudPut                StatsType = "Cloud provider put"
	TotalPutCalls           StatsType = "Number of cloud provider PUT"
	TotalGetCalls           StatsType = "Number of cloud provider GET"
	TotalAssignedIDsCreated StatsType = "Number of assigned ids created in this sync cycle"
	TotalAssignedIDsUpdated StatsType = "Number of assigned ids updated in this sync cycle"
	TotalAssignedIDsDeleted StatsType = "Number of assigned ids deleted in this sync cycle"
	K8sGet                  StatsType = "K8s get"
	K8sPut                  StatsType = "K8s put"
	FindAssignedIDDel       StatsType = "Find assigned ids to delete"
	FindAssignedIDCreate    StatsType = "Find assigned ids to create"
	AssignedIDDel           StatsType = "Assigned ID deletion"
	AssignedIDAdd           StatsType = "Assigned ID addition"
	TotalIDDel              StatsType = "Total time to delete assigned IDs"
	TotalIDAdd              StatsType = "Total time to add assigned IDs"
	TotalCreateOrUpdate     StatsType = "Total time to assign or remove IDs"

	EventRecord StatsType = "Event recording"
)

func Init() {
	GlobalStats = make(map[StatsType]time.Duration)
	CountStats = make(map[StatsType]int)
	Mutex = &sync.RWMutex{}
}

func Put(key StatsType, val time.Duration) {
	if GlobalStats != nil {
		Mutex.Lock()
		defer Mutex.Unlock()
		GlobalStats[key] = val
	}
}

func Get(key StatsType) time.Duration {
	if GlobalStats != nil {
		Mutex.RLock()
		defer Mutex.RUnlock()
		return GlobalStats[key]
	}
	return 0
}

func Update(key StatsType, val time.Duration) {
	if GlobalStats != nil {
		Mutex.Lock()
		defer Mutex.Unlock()
		GlobalStats[key] = GlobalStats[key] + val
	}
}

func Print(key StatsType) {
	Mutex.RLock()
	defer Mutex.RUnlock()

	klog.Infof("%s: %s", key, GlobalStats[key])
}

func PrintCount(key StatsType) {
	Mutex.RLock()
	defer Mutex.RUnlock()

	klog.Infof("%s: %d", key, CountStats[key])
}

func UpdateCount(key StatsType, val int) {
	Mutex.Lock()
	defer Mutex.Unlock()
	CountStats[key] = CountStats[key] + val
}

func PrintSync() {
	klog.Infof("** Stats collected **")
	if GlobalStats != nil {
		//first we list the
		Print(PodList)
		Print(IDList)
		Print(BindingList)
		Print(AssignedIDList)
		Print(System)
		Print(CacheSync)

		Print(CloudGet)
		Print(CloudPut)
		Print(AssignedIDAdd)
		Print(AssignedIDDel)

		PrintCount(TotalPutCalls)
		PrintCount(TotalGetCalls)

		PrintCount(TotalAssignedIDsCreated)
		PrintCount(TotalAssignedIDsUpdated)
		PrintCount(TotalAssignedIDsDeleted)

		Print(FindAssignedIDCreate)
		Print(FindAssignedIDDel)

		Print(TotalCreateOrUpdate)

		Print(EventRecord)
		Print(Total)
	}
	klog.Infof("*********************")
}

func GetAll() map[StatsType]time.Duration {
	Mutex.RLock()
	defer Mutex.RUnlock()
	return GlobalStats
}
