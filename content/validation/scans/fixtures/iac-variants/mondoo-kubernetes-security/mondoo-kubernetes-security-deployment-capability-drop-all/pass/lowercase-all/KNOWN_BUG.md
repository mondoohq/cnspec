dropsAllCapabilities matches only uppercase ALL, but containerd and CRI-O drop every capability for a lowercase all: mondoohq/mql#10769

The rollup in providers/k8s/resources/workload_security.go compares `d == "ALL"`. Once a provider with the strings.EqualFold fix is the one this suite resolves against, this scenario passes and the marker must be deleted. See mondoohq/cnspec#3660.
