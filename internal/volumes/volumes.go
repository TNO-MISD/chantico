package volumes

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
)

const (
	ChanticoVolumeMount       = "chantico-volume-mount"
	ChanticoVolumeLocationEnv = "CHANTICOVOLUMELOCATIONENV"
	ChanticoVolumeClaimEnv    = "CHANTICOVOLUMECLAIMENV"
)

func GetChanticoVolume() (corev1.Volume, error) {
	volumeClaim := os.Getenv(ChanticoVolumeClaimEnv)
	if volumeClaim == "" {
		return corev1.Volume{}, fmt.Errorf("environment variable %s is not set", ChanticoVolumeClaimEnv)
	}
	return corev1.Volume{
		Name: ChanticoVolumeMount,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: volumeClaim,
			},
		},
	}, nil

}
