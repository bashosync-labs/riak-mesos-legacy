package common

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	"math/rand"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewReservedScalarResource(name string, value float64, principal *string, role *string) *mesos.Resource {
	reservation := &mesos.Resource_ReservationInfo{}
	if principal != nil {
		reservation.Principal = principal
	}

	resource := util.NewScalarResource(name, value)
	resource.Role = role
	resource.Reservation = reservation
	return resource
}

func NewReservedVolume(value float64, containerPath *string, persistenceID *string, principal *string, role *string) *mesos.Resource {
	resource := NewReservedScalarResource("disk", value, principal, role)

	mode := mesos.Volume_RW
	volume := &mesos.Volume{
		ContainerPath: containerPath,
		Mode:          &mode,
	}
	persistence := &mesos.Resource_DiskInfo_Persistence{
		Id: persistenceID,
	}
	info := &mesos.Resource_DiskInfo{
		Persistence: persistence,
		Volume:      volume,
	}
	resource.Disk = info

	return resource
}

func CopyReservedVolumes(immutableResources []*mesos.Resource) []*mesos.Resource {
	reserved := []*mesos.Resource{}
	for _, resource := range FilterReservedVolumes(immutableResources) {
		containerPath := resource.Disk.Volume.GetContainerPath()
		persistenceID := resource.Disk.Persistence.GetId()
		principal := resource.Reservation.GetPrincipal()
		role := resource.GetRole()
		newVolume := NewReservedVolume(resource.Scalar.GetValue(), &containerPath, &persistenceID, &principal, &role)
		reserved = append(reserved, newVolume)
	}
	return reserved
}

func CopyReservedResources(immutableResources []*mesos.Resource) []*mesos.Resource {
	reserved := []*mesos.Resource{}
	for _, resource := range FilterReservedResources(immutableResources) {
		principal := resource.Reservation.GetPrincipal()
		role := resource.GetRole()
		newResource := NewReservedScalarResource(resource.GetName(), resource.Scalar.GetValue(), &principal, &role)
		reserved = append(reserved, newResource)
	}
	return reserved
}

func FilterReservedVolumes(immutableResources []*mesos.Resource) []*mesos.Resource {
	return util.FilterResources(immutableResources, func(res *mesos.Resource) bool {
		if res.Reservation != nil &&
			res.Disk != nil &&
			res.GetName() == "disk" {
			return true
		}
		return false
	})
}

func FilterReservedResources(immutableResources []*mesos.Resource) []*mesos.Resource {
	return util.FilterResources(immutableResources, func(res *mesos.Resource) bool {
		return res.Reservation != nil && res.GetRole() != ""
	})
}

func FilterUnreservedResources(immutableResources []*mesos.Resource) []*mesos.Resource {
	return util.FilterResources(immutableResources, func(res *mesos.Resource) bool {
		return res.Reservation == nil
	})
}

func ApplyScalarResources(mutableResources []*mesos.Resource, cpus float64, mem float64, disk float64) []*mesos.Resource {
	mutableResources = ApplyScalarResource(mutableResources, "cpus", cpus)
	mutableResources = ApplyScalarResource(mutableResources, "mem", mem)
	mutableResources = ApplyScalarResource(mutableResources, "disk", disk)
	return mutableResources
}

func ApplyRangesResource(mutableResources []*mesos.Resource, portCount int) ([]*mesos.Resource, *mesos.Resource) {
	leftoverPorts, extractedPorts := CreatePortsResourceFromResources(mutableResources, portCount)

	for idx := range util.FilterResources(mutableResources, func(res *mesos.Resource) bool { return res.GetName() == "ports" }) {
		mutableResources[idx] = leftoverPorts
	}
	return mutableResources, extractedPorts
}

func ApplyScalarResource(mutableResources []*mesos.Resource, name string, value float64) []*mesos.Resource {
	found := false
	for _, resource := range mutableResources {
		if resource.GetName() == name && !found {
			newValue := resource.GetScalar().GetValue() - value
			resource.Scalar.Value = &newValue
			found = true
		}
	}
	// for _, resource := range util.FilterResources(mutableResources, func(res *mesos.Resource) bool { return res.GetName() == name }) {
	// 	newValue := resource.GetScalar().GetValue() - value
	// 	resource.Scalar.Value = &newValue
	// }
	return mutableResources
}

func PortResourceWillFit(immutableResources []*mesos.Resource, portCount int) bool {
	for _, resource := range util.FilterResources(immutableResources, func(res *mesos.Resource) bool { return res.GetName() == "ports" }) {
		ports := RangesToArray(resource.GetRanges().GetRange())
		if len(ports) >= portCount {
			return true
		}
	}
	log.Info("Ports didn't fit")
	return false
}

func ScalarResourcesWillFit(immutableResources []*mesos.Resource, cpus float64, mem float64, disk float64) bool {
	log.Infof("Requested: cpus:%v, mem:%v, disk:%v", cpus, mem, disk)
	log.Infof("Checking to see if these resources fit: %s", PrettyStringForScalarResources(immutableResources))

	if !ScalarResourceWillFit(immutableResources, "cpus", cpus) {
		log.Info("CPU didn't fit")
		return false
	}
	if !ScalarResourceWillFit(immutableResources, "mem", mem) {
		log.Info("Mem didn't fit")
		return false
	}
	if !ScalarResourceWillFit(immutableResources, "disk", disk) {
		log.Info("Disk didn't fit")
		return false
	}

	return true
}

func ScalarResourceWillFit(immutableResources []*mesos.Resource, name string, value float64) bool {
	for _, resource := range util.FilterResources(immutableResources, func(res *mesos.Resource) bool { return res.GetName() == name }) {
		if resource.GetScalar().GetValue() >= (value) {
			return true
		}
	}
	return false
}

func PrettyStringForScalarResources(resources []*mesos.Resource) string {
	output := "["
	for _, resource := range resources {
		if resource.Scalar != nil {
			if resource.Reservation != nil {
				output += "R("
			} else {
				output += "U("
			}
			output += fmt.Sprintf("%s:%v), ", resource.GetName(), resource.Scalar.GetValue())
		}
	}
	output += "]"
	return output
}

func PortIterator(resources []*mesos.Resource) chan int64 {
	ports := make(chan int64)
	go func() {
		defer close(ports)
		for _, resource := range util.FilterResources(resources, func(res *mesos.Resource) bool { return res.GetName() == "ports" }) {
			for _, port := range RangesToArray(resource.GetRanges().GetRange()) {
				ports <- port
			}
		}
	}()
	return ports
}

func CreatePortsResourceFromResources(immutableResources []*mesos.Resource, portCount int) (*mesos.Resource, *mesos.Resource) {
	leftoverPortsResource := &mesos.Resource{}
	ask := &mesos.Resource{}

	for _, resource := range util.FilterResources(immutableResources, func(res *mesos.Resource) bool { return res.GetName() == "ports" }) {
		ports := RangesToArray(resource.GetRanges().GetRange())

		sliceLoc := 0
		if len(ports)-portCount > 0 {
			sliceLoc = rand.Intn(len(ports) - portCount)
		}
		takingPorts := make([]int64, portCount)
		copy(takingPorts, ports[sliceLoc:(sliceLoc+portCount)])
		leavingPorts := make([]int64, len(ports)-portCount)
		copy(leavingPorts, ports[:sliceLoc])
		copy(leavingPorts[sliceLoc:], ports[(sliceLoc+portCount):])
		leftoverPortsResource = util.NewRangesResource("ports", ArrayToRanges(leavingPorts))
		ask := util.NewRangesResource("ports", ArrayToRanges(takingPorts))
		return leftoverPortsResource, ask
	}

	return leftoverPortsResource, ask
}

func RemoveReservations(resources []*mesos.Resource) []*mesos.Resource {
	for _, resource := range resources {
		resource.Reservation = nil
		resource.Disk = nil
		resource.Role = nil
	}

	return resources
}

type intarray []int64

func (a intarray) Len() int           { return len(a) }
func (a intarray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a intarray) Less(i, j int) bool { return a[i] < a[j] }

type ResourceAsker (func([]*mesos.Resource) (remaining []*mesos.Resource, ask *mesos.Resource, success bool))

// We assume the input is sorted
func ArrayToRanges(ports []int64) []*mesos.Value_Range {
	sort.Sort(intarray(ports))
	if len(ports) == 0 {
		return []*mesos.Value_Range{}
	}
	fakeret := [][]int64{[]int64{ports[0], ports[0]}}
	for _, val := range ports {
		if val > fakeret[len(fakeret)-1][1]+1 {
			fakeret = append(fakeret, []int64{val, val})
		} else {
			fakeret[len(fakeret)-1][1] = val
		}
	}
	ret := make([]*mesos.Value_Range, len(fakeret))
	for idx := range fakeret {
		ret[idx] = util.NewValueRange(uint64(fakeret[idx][0]), uint64(fakeret[idx][1]))
	}
	return ret
}
func RangesToArray(ranges []*mesos.Value_Range) []int64 {
	array := make([]int64, 0)
	for _, mesosRange := range ranges {
		temp := make([]int64, mesosRange.GetEnd()-mesosRange.GetBegin()+1)
		idx := 0
		for i := mesosRange.GetBegin(); i <= mesosRange.GetEnd(); i++ {
			temp[idx] = int64(i)
			idx++
		}
		array = append(array, temp...)
	}
	return array
}

func KillEPMD(dir string) error {
	globs, err := filepath.Glob(filepath.Join(dir, "/erts-*/bin/epmd"))
	if err != nil {
		return err
	}
	if len(globs) != 1 {
		return errors.New(fmt.Sprintf("Not the right number of globs: %d", len(globs)))
	}

	cpEPMDCmd := exec.Command("/bin/cp", "/bin/true", globs[0])
	err = cpEPMDCmd.Run()
	if err != nil {
		return err
	}
	return nil
}
