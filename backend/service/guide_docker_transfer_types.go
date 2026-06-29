package service

import dockertransfer "github.com/linkease/quick-start/istore-backend/modules/guidestorage/dockertransfer"

type GuideDockerRootSnapshot struct {
	Path string
}

type GuideDockerPartitionCandidate = dockertransfer.PartitionCandidate
