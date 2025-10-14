package volumes

import (
	"os"
	"testing"
)

func TestGetChanticoVolume(t *testing.T) {
	os.Setenv(ChanticoVolumeClaimEnv, "test")
	volume, err := GetChanticoVolume()
	if err == nil && volume.VolumeSource.PersistentVolumeClaim.ClaimName != "test" {
		t.Errorf("%#v is not in sync with the volume definition %#v", ChanticoVolumeClaimEnv, &volume.VolumeSource.PersistentVolumeClaim)
	}
	os.Unsetenv(ChanticoVolumeClaimEnv)
}
