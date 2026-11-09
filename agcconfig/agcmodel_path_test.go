package agcconfig

import "testing"

func TestBuildAgcParamPathKeepsStationRoot(t *testing.T) {
	got := buildAgcParamPath(
		"/station/station/10kV1进线/t1/pcs1/bms",
		"ANALOG",
		"SOC",
	)
	want := "station/10kV1进线/t1/pcs1/bms/ANALOG/SOC"
	if got != want {
		t.Fatalf("buildAgcParamPath() = %q, want %q", got, want)
	}
}

func TestSetSubDevicePathsProducesAgcCompatiblePath(t *testing.T) {
	root := &Device{Name: "station", Type: "STATION"}
	inline := &Device{Name: "qa_ems_model_test", Type: "INLINE"}
	bms := &Device{Name: "qa_ems_model_test_bms", Type: "BMS"}

	if err := inline.addSubDevice(bms); err != nil {
		t.Fatalf("add BMS: %v", err)
	}
	if err := root.addSubDevice(inline); err != nil {
		t.Fatalf("add INLINE: %v", err)
	}
	root.setSubDevicePaths()

	got := buildAgcParamPath(bms.Path, "ANALOG", "SOC")
	want := "station/qa_ems_model_test/qa_ems_model_test_bms/ANALOG/SOC"
	if got != want {
		t.Fatalf("repaired path = %q, want %q", got, want)
	}
}
