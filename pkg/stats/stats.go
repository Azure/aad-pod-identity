package stats

import (
	"sync"
	"time"

	"k8s.io/klog"
)

var GlobalStats map[Type]time.Duration
var CountStats map[Type]int

var Mutex *sync.RWMutex

type Type string

const (
	Total                   Type = "Total"
	System                  Type = "System"
	CacheSync               Type = "CacheSync"
	CurrentState            Type = "Gather current state"
	PodList                 Type = "Pod listing"
	BindingList             Type = "Binding listing"
	IDList                  Type = "ID listing"
	ExceptionList           Type = "Pod Identity Exception listing"
	AssignedIDList          Type = "Assigned ID listing"
	CloudGet                Type = "Cloud provider get"
	CloudPut                Type = "Cloud provider put"
	TotalPutCalls           Type = "Number of cloud provider PUT"
	TotalGetCalls           Type = "Number of cloud provider GET"
	TotalAssignedIDsCreated Type = "Number of assigned ids created in this sync cycle"
	TotalAssignedIDsUpdated Type = "Number of assigned ids updated in this sync cycle"
	TotalAssignedIDsDeleted Type = "Number of assigned ids deleted in this sync cycle"
	K8sGet                  Type = "K8s get"
	K8sPut                  Type = "K8s put"
	FindAssignedIDDel       Type = "Find assigned ids to delete"
	FindAssignedIDCreate    Type = "Find assigned ids to create"
	AssignedIDDel           Type = "Assigned ID deletion"
	AssignedIDAdd           Type = "Assigned ID addition"
	TotalIDDel              Type = "Total time to delete assigned IDs"
	TotalIDAdd              Type = "Total time to add assigned IDs"
	TotalCreateOrUpdate     Type = "Total time to assign or remove IDs"

	EventRecord Type = "Event recording"
)

func Init() {
	GlobalStats = make(map[Type]time.Duration)
	CountStats = make(map[Type]int)
	Mutex = &sync.RWMutex{}
}

func Put(key Type, val time.Duration) {
	if GlobalStats != nil {
		Mutex.Lock()
		defer Mutex.Unlock()
		GlobalStats[key] = val
	}
}

func Get(key Type) time.Duration {
	if GlobalStats != nil {
		Mutex.RLock()
		defer Mutex.RUnlock()
		return GlobalStats[key]
	}
	return 0
}

func Update(key Type, val time.Duration) {
	if GlobalStats != nil {
		Mutex.Lock()
		defer Mutex.Unlock()
		GlobalStats[key] = GlobalStats[key] + val
	}
}

func Print(key Type) {
	Mutex.RLock()
	defer Mutex.RUnlock()

	klog.Infof("%s: %s", key, GlobalStats[key])
}

func PrintCount(key Type) {
	Mutex.RLock()
	defer Mutex.RUnlock()

	klog.Infof("%s: %d", key, CountStats[key])
}

func UpdateCount(key Type, val int) {
	Mutex.Lock()
	defer Mutex.Unlock()
	CountStats[key] = CountStats[key] + val
}

func PrintSync() {
	klog.Infof("** stats collected **")
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

func GetAll() map[Type]time.Duration {
	Mutex.RLock()
	defer Mutex.RUnlock()
	return GlobalStats
}
