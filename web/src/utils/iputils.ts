import ipaddr from "ipaddr.js"

const _ipv4 = /\d+\.\d+\.\d+\.\d+/

// 是否有效ip
const isValidIPv4 = (ip: string): boolean => {
    return _ipv4.test(ip) && ipaddr.IPv4.isValid(ip)
}
const ipToLong = (ip: string): number => {
    const array = ipaddr.IPv4.parse(ip).toByteArray();
    return (array[0] << 24) | (array[1] << 16) | (array[2] << 8) | array[3];
}
const ipFromLong = (num: number): string => {
    return ipaddr.fromByteArray([(num >> 24) & 0xff, (num >> 16) & 0xff, (num >> 8) & 0xff, num & 0xff]).toString()
}
// 是否是子网掩码
const isValidMask = (ipmask: string): boolean => {

    if (!_ipv4.test(ipmask) || !ipaddr.IPv4.isIPv4(ipmask)) {
        return false;
    }
    let tmp = 0;
    let mask = ipToLong(ipmask);
    for (let j = 31; j >= 0; j--) {
        //console.log("1<<j", 1<<j, tmp, j, ~tmp);
        if (0 == (mask & (1 << j))) {
            break;
        }
        tmp = tmp + (1 << j);
    }
    if (((~tmp) & mask) != 0) {
        return false
    }

    return true;
}
// 是否在合法的子网掩码范围
const isValidMaskRange = (ipStr: string, maskStr: string, rangeStart: string, rangeEnd: string): boolean => {
    let ipStartLong = ipToLong(ipStr) & ipToLong(maskStr);
    let rangeStartLong = ipToLong(rangeStart);
    let rangeEndLong = ipToLong(rangeEnd);
    let maskLong = ipToLong(maskStr);
    let ipPoolSize = (~maskLong);
    if (rangeStartLong < rangeEndLong && rangeStartLong > (ipStartLong + 1) && rangeEndLong < (ipStartLong + ipPoolSize)) {
        return true;
    }
    return false;
}
// 计算子网掩码范围
const calcMaskRange = (ipStr: string, maskStr: string): string[] => {
    let maskLong = ipToLong(maskStr);
    let ipStartLong = ipToLong(ipStr) & maskLong;
    let ipPoolSize = (~maskLong);
    let ipPoolEnd;
    if (ipPoolSize >= 105) {
        ipPoolEnd = ipStartLong | (ipPoolSize - 5);
        ipStartLong = ipStartLong | 100;
    } else if (ipPoolSize >= 3) {
        ipPoolEnd = ipStartLong | (ipPoolSize - 1);
        ipStartLong = ipStartLong | 2;
    } else {
        ipStartLong = ipStartLong | 1;
        ipPoolEnd = ipStartLong;
    }
    return [ipFromLong(ipStartLong), ipFromLong(ipPoolEnd)];
}
// 前缀长度转掩码
const prefixToMask = (prefix: number): string => {
    return ipaddr.IPv4.subnetMaskFromPrefixLength(prefix).toString()
}


export default {
    isValidMask,
    isValidIPv4,
    isValidMaskRange,
    calcMaskRange,
    prefixToMask
}


// 示例
// import iper from "./iptest"
// console.log("255.255.255.0", isValidMask("255.255.255.0"));
// console.log("255.255.0.0", isValidMask("255.255.0.0"));
// console.log("255.0.0.0", isValidMask("255.0.0.0"));
// console.log("255.240.0.0", isValidMask("255.240.0.0"));
// console.log("255.241.0.0", isValidMask("255.241.0.0"));
// console.log("255.255.1.0", isValidMask("255.255.1.0"));
// console.log("255.255.0", isValidMask("255.255.1.0"));
// console.log("255.0.255.0", isValidMask("255.0.255.0"));
// console.log("=192.168.50.1", isValidIPv4("192.168.50.1"));
// console.log("=192.168.50.256", isValidIPv4("192.168.50.256"));
// console.log("192.168.50.1",isLocalIp("192.168.50.1"));

// const maskRangeTest = (ipStr: string, maskStr: string) => {
//     let iprange = calcMaskRange(ipStr, maskStr);
//     console.log("input", ipStr, maskStr, "output", iprange[0], iprange[1]);
// }

// const maskValidTest = (ipStr: string, maskStr: string, rangeStart: string, rangeEnd: string, should: boolean) => {
//     let ret = isValidMaskRange(ipStr, maskStr, rangeStart, rangeEnd);
//     if (ret === should) {
//         //  ok
//         console.log("OK", ipStr, maskStr, rangeStart, rangeEnd, "==>", ret);
//     } else {
//         console.log("failed", ipStr, maskStr, rangeStart, rangeEnd, "==>", ret, should);
//     }
// }

// maskRangeTest("192.168.9.1", "255.255.255.0")

// maskRangeTest("192.168.9.1", "255.255.0.0")

// maskValidTest("192.168.9.1", "255.255.255.0", "192.168.9.100", "192.168.9.250", true)
