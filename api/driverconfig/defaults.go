package driverconfig

const (
	// DefaultKubeletRootDir is the standard kubelet root directory, so behaviour is
	// unchanged unless the kubelet --root-dir is relocated.
	DefaultKubeletRootDir = "/var/lib/kubelet"

	DefaultBindAddress = ":8080"
)
