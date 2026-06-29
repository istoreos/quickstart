package dockertransfer

import "errors"

type PathSnapshot struct {
	TargetPath    string
	OriginPath    string
	TargetSource  string
	TargetFSType  string
	OverlaySource string
	OriginSource  string
}

func ValidatePathSnapshot(snapshot PathSnapshot) error {
	if snapshot.TargetPath == snapshot.OriginPath {
		return errors.New("不能选择同一个目录")
	}
	if snapshot.TargetFSType == "overlay" || snapshot.TargetFSType == "tmpfs" {
		return errors.New("路径不合法，不能在系统目录或者mnt根目录上")
	}
	if snapshot.TargetFSType == "ntfs" {
		return errors.New("路径不合法，不能在ntfs分区的目录上")
	}
	if snapshot.OverlaySource == snapshot.TargetSource {
		return errors.New("路径不合法，不能在系统目录上")
	}
	if snapshot.OriginSource == snapshot.TargetSource {
		return errors.New("不能选择原docker根目录所在分区")
	}
	return nil
}
