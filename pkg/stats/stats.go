package stats

import (
	"sync"
	"time"

	"k8s.io/klog"
)

var (
	// GlobalStats is a map that stores the duration of each stats.
	GlobalStats map[Type]time.Duration

	// CountStats is a map that stores the count of each stats.
	CountStats map[Type]int

	mutex *sync.RWMutex
)

// Type represents differnet statistics that are being collected.
type Type string

const (
	// Total represents the total duration of a specific operation.
	Total Type = "Total"

	// System represents the duration it takes to list all aad-pod-identity CRDs.
	System Type = "System"

	// CacheSync represents the duration it takes to sync CRD client's cache.
	CacheSync Type = "CacheSync"

	// CurrentState represents the duration it takes to generate a list of desired AzureAssignedIdentities.
	CurrentState Type = "Gather current state"

	// PodList represents the duration it takes to list pods.
	PodList Type = "Pod listing"

	// BindingList represents the duration it takes to list AzureIdentityBindings.
	BindingList Type = "Binding listing"

	// IDList represents the duration it takes to list AzureIdentities.
	IDList Type = "ID listing"

	// ExceptionList represents the duration it takes to list AzurePodIdentityExceptions.
	ExceptionList Type = "Pod Identity Exception listing"

	// AssignedIDList represents the duration it takes to list AzureAssignedIdentities.
	AssignedIDList Type = "Assigned ID listing"

	// CloudGet represents the duration it takes to complete a GET request to ARM in a given sync cycle.
	CloudGet Type = "Cloud provider get"

	// CloudUpdate represents the duration it takes to complete a PATCH request to ARM in a given sync cycle.
	CloudUpdate Type = "Cloud provider update"

	// TotalUpdateCalls represents the number of PATCH requests to ARM in a given sync cycle.
	TotalUpdateCalls Type = "Number of cloud provider PATCH"

	// TotalGetCalls represents the number of GET requests to ARM in a given sync cycle.
	TotalGetCalls Type = "Number of cloud provider GET"

	// TotalAssignedIDsCreated represents the number of AzureAssignedIdentities created in a given sync cycle.
	TotalAssignedIDsCreated Type = "Number of assigned ids created in this sync cycle"

	// TotalAssignedIDsUpdated represents the number of AzureAssignedIdentities updated in a given sync cycle.
	TotalAssignedIDsUpdated Type = "Number of assigned ids updated in this sync cycle"

	// TotalAssignedIDsDeleted represents the number of AzureAssignedIdentities deleted in a given sync cycle.
	TotalAssignedIDsDeleted Type = "Number of assigned ids deleted in this sync cycle"

	// FindAssignedIDDel represents the duration it takes to generate a list of AzureAssignedIdentities to be deleted.
	FindAssignedIDDel Type = "Find assigned ids to delete"

	// FindAssignedIDCreate represents the duration it takes to generate a list of AzureAssignedIdentities to be created.
	FindAssignedIDCreate Type = "Find assigned ids to create"

	// AssignedIDDel represents the duration it takes to delete an AzureAssignedIdentity.
	AssignedIDDel Type = "Assigned ID deletion"

	// AssignedIDAdd represents the duration it takes to create an AzureAssignedIdentity.
	AssignedIDAdd Type = "Assigned ID addition"

	// TotalCreateOrUpdate represents the duration it takes to create or update a given list of AzureAssignedIdentities.
	TotalCreateOrUpdate Type = "Total time to assign or update IDs"
)

// Init initializes the maps uesd to store the stats.
func Init() {
	GlobalStats = make(map[Type]time.Duration)
	CountStats = make(map[Type]int)
	mutex = &sync.RWMutex{}
}

// Put puts a value to a specific stat.
func Put(key Type, val time.Duration) {
	if GlobalStats != nil {
		mutex.Lock()
		defer mutex.Unlock()
		GlobalStats[key] = val
	}
}

// Get returns the stat value of a given key.
func Get(key Type) time.Duration {
	if GlobalStats != nil {
		mutex.RLock()
		defer mutex.RUnlock()
		return GlobalStats[key]
	}
	return 0
}

// Update updates the value of a specific stat.
func Update(key Type, val time.Duration) {
	if GlobalStats != nil {
		mutex.Lock()
		defer mutex.Unlock()
		GlobalStats[key] = GlobalStats[key] + val
	}
}

// Print prints the value of a specific stat.
func Print(key Type) {
	mutex.RLock()
	defer mutex.RUnlock()

	klog.Infof("%s: %s", key, GlobalStats[key])
}

// PrintCount prints the count of a specific stat.
func PrintCount(key Type) {
	mutex.RLock()
	defer mutex.RUnlock()

	klog.Infof("%s: %d", key, CountStats[key])
}

// UpdateCount updates the count of a specific stat.
func UpdateCount(key Type, val int) {
	mutex.Lock()
	defer mutex.Unlock()
	CountStats[key] = CountStats[key] + val
}

// PrintSync prints all relevant statistics in a sync cycle.
func PrintSync() {
	klog.Infof("** stats collected **")
	if GlobalStats != nil {
		Print(PodList)
		Print(IDList)
		Print(BindingList)
		Print(AssignedIDList)
		Print(System)
		Print(CacheSync)

		Print(CloudGet)
		Print(CloudUpdate)
		Print(AssignedIDAdd)
		Print(AssignedIDDel)

		PrintCount(TotalUpdateCalls)
		PrintCount(TotalGetCalls)

		PrintCount(TotalAssignedIDsCreated)
		PrintCount(TotalAssignedIDsUpdated)
		PrintCount(TotalAssignedIDsDeleted)

		Print(FindAssignedIDCreate)
		Print(FindAssignedIDDel)

		Print(TotalCreateOrUpdate)

		Print(Total)
	}
	klog.Infof("*********************")
}

// GetAll returns the global statistics it is currently collecting
func GetAll() map[Type]time.Duration {
	mutex.RLock()
	defer mutex.RUnlock()
	return GlobalStats
}
