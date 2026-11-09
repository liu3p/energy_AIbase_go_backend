package agcconfig

import (
	"errors"
	"testing"
)

func TestUpdateAgcDeviceRenamesAndRefreshesPaths(t *testing.T) {
	station := &Device{
		Name: "station",
		Type: "STATION",
		SubDevices: []*Device{
			{
				Name: "old_inline",
				Type: "INLINE",
				Para: []Para{
					{Name: "Capacity", Type: "ANALOG"},
				},
				DynPara: []DynPara{
					{Name: "P", Type: "ANALOG"},
				},
				SubDevices: []*Device{
					{Name: "bms", Type: "BMS"},
				},
			},
		},
	}
	station.setSubDevicePaths()
	manager := &AgcModelManager{station: station}
	persistCalls := 0

	updated, err := manager.updateAgcDeviceWithPersist(
		"old_inline",
		`{"name":"renamed_inline","type":"INLINE","para":[],"dyn_para":[]}`,
		func() error {
			persistCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("rename failed: %v", err)
	}
	if updated.Name != "renamed_inline" {
		t.Fatalf("updated name = %q, want renamed_inline", updated.Name)
	}
	if _, ok := station.getSubDeviceByName("renamed_inline"); !ok {
		t.Fatal("renamed device cannot be found")
	}
	if _, ok := station.getSubDeviceByName("old_inline"); ok {
		t.Fatal("old device name still exists")
	}
	renamed, _ := station.getSubDeviceByName("renamed_inline")
	if renamed.Path != "/station/station/renamed_inline" {
		t.Fatalf("renamed path = %q", renamed.Path)
	}
	if renamed.SubDevices[0].Path != "/station/station/renamed_inline/bms" {
		t.Fatalf("child path = %q", renamed.SubDevices[0].Path)
	}
	if renamed.Para[0].Path != "/station/station/renamed_inline/ANALOG/Capacity" {
		t.Fatalf("parameter path = %q", renamed.Para[0].Path)
	}
	if renamed.DynPara[0].Path != "/station/station/renamed_inline/ANALOG/P" {
		t.Fatalf("dynamic parameter path = %q", renamed.DynPara[0].Path)
	}
	if persistCalls != 1 {
		t.Fatalf("persist calls = %d, want 1", persistCalls)
	}
}

func TestUpdateAgcDeviceRejectsDuplicateRename(t *testing.T) {
	station := &Device{
		Name: "station",
		Type: "STATION",
		SubDevices: []*Device{
			{Name: "old_inline", Type: "INLINE"},
			{Name: "existing_inline", Type: "INLINE"},
		},
	}
	station.setSubDevicePaths()
	manager := &AgcModelManager{station: station}
	persistCalls := 0

	_, err := manager.updateAgcDeviceWithPersist(
		"old_inline",
		`{"name":"existing_inline","type":"INLINE","para":[],"dyn_para":[]}`,
		func() error {
			persistCalls++
			return nil
		},
	)
	if err == nil {
		t.Fatal("duplicate rename should fail")
	}
	if persistCalls != 0 {
		t.Fatalf("persist calls = %d, want 0", persistCalls)
	}
	if _, ok := station.getSubDeviceByName("old_inline"); !ok {
		t.Fatal("original device changed after rejected rename")
	}
}

func TestUpdateAgcDeviceRestoresNameWhenPersistFails(t *testing.T) {
	station := &Device{
		Name: "station",
		Type: "STATION",
		SubDevices: []*Device{
			{Name: "old_inline", Type: "INLINE"},
		},
	}
	station.setSubDevicePaths()
	manager := &AgcModelManager{station: station}
	persistCalls := 0

	_, err := manager.updateAgcDeviceWithPersist(
		"old_inline",
		`{"name":"renamed_inline","type":"INLINE","para":[],"dyn_para":[]}`,
		func() error {
			persistCalls++
			if persistCalls == 1 {
				return errors.New("forced persist failure")
			}
			return nil
		},
	)
	if err == nil {
		t.Fatal("persistence failure should be returned")
	}
	if persistCalls != 2 {
		t.Fatalf("persist calls = %d, want rollback attempt", persistCalls)
	}
	if _, ok := station.getSubDeviceByName("old_inline"); !ok {
		t.Fatal("old name was not restored")
	}
	if _, ok := station.getSubDeviceByName("renamed_inline"); ok {
		t.Fatal("new name remained after failed persistence")
	}
}
