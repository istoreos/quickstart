package ubus

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[UbusStatusOK-0]
	_ = x[UbusStatusInvalidCommand-1]
	_ = x[UbusStatusInvalidArgument-2]
	_ = x[UbusStatusMethodNotFound-3]
	_ = x[UbusStatusNotFound-4]
	_ = x[UbusStatusNoData-5]
	_ = x[UbusStatusPermissionDenied-6]
	_ = x[UbusStatusTimeout-7]
	_ = x[UbusStatusNotSupported-8]
	_ = x[UbusStatusUnknownError-9]
	_ = x[UbusStatusConnectionFailed-10]
	_ = x[UbusStatusLast-11]
}

const _UbusResponseCode_name = "UbusStatusOKUbusStatusInvalidCommandUbusStatusInvalidArgumentUbusStatusMethodNotFoundUbusStatusNotFoundUbusStatusNoDataUbusStatusPermissionDeniedUbusStatusTimeoutUbusStatusNotSupportedUbusStatusUnknownErrorUbusStatusConnectionFailedUbusStatusLast"

var _UbusResponseCode_index = [...]uint8{0, 12, 36, 61, 85, 103, 119, 145, 162, 184, 206, 232, 246}

func (i UbusResponseCode) String() string {
	if i < 0 || i >= UbusResponseCode(len(_UbusResponseCode_index)-1) {
		return "UbusResponseCode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _UbusResponseCode_name[_UbusResponseCode_index[i]:_UbusResponseCode_index[i+1]]
}
