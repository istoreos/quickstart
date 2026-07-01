import { Do } from "./do";
import { useGettextLazy, formatNumber } from "/@/plugins/i18n";
const { $gettext, $ngettext } = useGettextLazy();

declare interface ResponseData<T = any> {
  success?: number;
  error?: string;
  scope?: string;
  detail?: string;
  result?: T;
}
const baseURL = "/cgi-bin/luci/istore";
const baseURL1 = "/cgi-bin/luci/admin";
let reloading = false;
const Request = <T>(url: string, data: RequestInit) => {
  if (url.indexOf("//") == -1) url = `${baseURL}${url}`;
  return Do<ResponseData<T>>(url, data).then((res) => {
    if (res?.data) {
      if (res.data.success == -1001 && res.data.error == "Forbidden") {
        if (!reloading) {
          reloading = true;
          alert($gettext("登录过期，请重新登录"));
          location.reload();
        }
      }
    }
    return res;
  });
};
const Request1 = <T>(url: string, data: RequestInit) => {
  if (url.indexOf("//") == -1) url = `${baseURL1}${url}`;
  return Do<ResponseData<T>>(url, data).then((res) => {
    if (res?.data) {
      if (res.data.success == -1001 && res.data.error == "Forbidden") {
        if (!reloading) {
          reloading = true;
          alert($gettext("登录过期，请重新登录"));
          location.reload();
        }
      }
    }
    return res;
  });
};
export const Network = {
  Statistics: {
    //网速统计
    GET() {
      return Request<NetworkStatistics>("/u/network/statistics/", {
        method: "GET",
      });
    },
  },
  Status: {
    //状态信息
    GET() {
      return Request<NetworkStatus>("/u/network/status/", {
        method: "GET",
      });
    },
  },
  Device: {
    //设备信息
    List: {
      GET() {
        return Request<NetworkDeviceList>("/network/device/list/", {
          method: "GET",
        });
      },
    },
  },
  Homebox: {
    //打开homebox服务
    Enable: {
      POST() {
        return Request<NetworkHomeboxEnable>("/network/homebox/enable/", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
        });
      },
    },
  },
  CheckPublickNet: {
    //检查设备是否有公网ip
    POST(data: { ipVersion: string }) {
      return Request<NetworkCheckPublickNet>("/network/checkPublicNet/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  GetInterfaceConfig: {
    //网络接口配置
    GET() {
      return Request<NetworkInterfaceGetConfigResponseResult>(
        "/network/interface/config/",
        {
          method: "GET",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
        }
      );
    },
  },

  POSTInterfaceConfig: {
    //网络接口配置
    POST(data: NetworkInterfaceSetConfigRequest) {
      return Request("/network/interface/config/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  PortList: {
    //查看网络接口状态
    GET() {
      return Request<NetworkPortList>("/network/port/list/", {
        method: "GET",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
      });
    },
  },
};
export const System = {
  Version: {
    //当前系统版本
    GET() {
      return Request<SystemVersion>("/u/system/version/", {
        method: "GET",
      });
    },
  },
  CheckUpdate: {
    GET() {
      return Request<SystemCheckUpdate>("/system/check-update/", {
        method: "GET",
      });
    },
  },
  AutoCheckUpdate: {
    POST(data: { enable: boolean }) {
      return Request<null>("/system/auto-check-update/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  Reboot: {
    POST() {
      return Request<null>("/system/reboot/", {
        method: "POST"
      });
    },
  },
  Status: {
    //当前系统版本
    GET() {
      return Request<SystemStatus>("/system/status/", {
        method: "GET",
      });
    },
  },
};
export const Nas = {
  Disk: {
    Status: {
      GET() {
        return Request<NasDiskStatus>(
          /**/ "/nas/disk/status/" /*/"http://127.0.0.1:8301/mock/nas_disk.json"/**/,
          {
            method: "GET",
          }
        );
      },
    },
    Erase: {
      POST(data: {
        devName: string; //设备名称
        path: string; //设备路径
        uuid: string; //设备uuid
        total: string; //设备容量
        mountPoint: string; //如已填写，则为已挂载点
      }) {
        return Request<null>("/nas/disk/erase", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
          body: JSON.stringify(data),
        });
      },
    },
    Init: {
      //初始化硬盘并挂载
      POST: (data: { name: string; path: string }) => {
        return Request<NasDiskModel>("/nas/disk/init/", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
          body: JSON.stringify(data),
        });
      },
    },
    InitRest: {
      //初始化硬盘剩余空间并挂载
      POST: (data: {
        name?: string | undefined;
        path?: string | undefined;
      }) => {
        return Request<NasDiskInitrest>("/nas/disk/initrest/", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
          body: JSON.stringify(data),
        });
      },
    },
    Partition: {
      Format: {
        //格式化已挂载分区
        POST: (data: { path: string; uuid?: string; mountPoint?: string }) => {
          return Request<MountPoint>("/nas/disk/partition/format", {
            method: "POST",
            headers: {
              "Content-Type": "application/json;charset=utf-8",
            },
            body: JSON.stringify(data),
          });
        },
      },
      Mount: {
        //挂载分区
        POST: (data: { path: string; uuid: string; mountPoint: string }) => {
          return Request<NasDiskPartitionMount>("/nas/disk/partition/mount", {
            method: "POST",
            headers: {
              "Content-Type": "application/json;charset=utf-8",
            },
            body: JSON.stringify(data),
          });
        },
      },
    },
  },
  Service: {
    Status: {
      GET() {
        return Request<NasServiceStatus>("/u/nas/service/status/", {
          method: "GET",
        });
      },
    },
  },
  Linkease: {
    Enable: {
      POST() {
        return Request<{
          port?: string;
        }>("/u/nas/linkease/enable", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
        });
      },
    },
  },
  Sandbox: {
    //沙箱向导
    POST(data: { path: string }) {
      return Request<null>("/nas/sandbox/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },

  GetSandbox: {
    //获取沙箱向导
    GET() {
      return Request<NasGetSandbox>("/nas/sandbox/", {
        method: "GET",
      });
    },
  },
  SandboxDisks: {
    //沙箱向导获取磁盘列表
    GET() {
      return Request<NasSandboxDisks>("/nas/sandbox/disks/", {
        method: "GET",
      });
    },
  },
  SandboxCommit: {
    //沙箱向导提交更改
    POST() {
      return Request<null>("/u/nas/sandbox/commit/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify({}),
      });
    },
  },
  SandboxReset: {
    //沙箱向导重置更改
    POST() {
      return Request<null>("/nas/sandbox/reset/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
      });
    },
  },
  SandboxExit: {
    //退出沙箱
    POST() {
      return Request<null>("/nas/sandbox/exit/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
      });
    },
  },
};
export const Share = {
  User: {
    List: {
      GET() {
        return Request<ShareUserListResponse>("/share/user/list/", {
          method: "GET",
        });
      },
    },
    Create: {
      POST(data: ShareUserCreateRequest) {
        return Request<null>("/share/user/create/", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
          body: JSON.stringify(data),
        });
      },
    },
    Update: {
      POST(data: ShareUserCreateRequest) {
        return Request<null>("/share/user/update/", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
          body: JSON.stringify(data),
        });
      },
    },
  },
  Service: {
    List: {
      GET() {
        return Request<ShareServiceListResponse>("/share/service/list/", {
          method: "GET",
        });
      },
    },
    Create: {
      POST(data: ShareServiceCreateRequest) {
        return Request<null>("/share/service/create/", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
          body: JSON.stringify(data),
        });
      },
    },
    Update: {
      POST(data: ShareServiceCreateRequest) {
        return Request<null>("/share/service/update/", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
          body: JSON.stringify(data),
        });
      },
    },
  },
};
export const App = {
  //插件相关接口
  Check: {
    POST(data: {
      //查看插件状态
      name: string; //插件名称
      checkRunning?: boolean;
    }) {
      return Request<AppCheckResult>("/app/check/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  Install: {
    POST(data: {
      name: string; //插件名称
    }) {
      return Request<null>("/app/install/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
};
export const Guide = {
  //向导相关接口
  Pppoe: {
    //pppoe拨号配置
    GET() {
      return Request<GuidePppoe>("/guide/pppoe/", {
        method: "GET",
      });
    },
    POST(data: { account: string; password: string }) {
      return Request<null>("/guide/pppoe/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  DnsConfig: {
    //dns配置
    GET() {
      return Request<GuideDnsConfig>("/guide/dns-config/", {
        method: "GET",
      });
    },
    POST(data: GuideDnsConfig) {
      return Request<null>("/guide/dns-config/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  DhcpClient: {
    //dhcp客户端模式
    POST(data: {
      wanProto: string; //static, dhcp WAN 接口配置方式
      dnsProto: string; //DNS 配置方式 static, dhcp
      staticIp: string; //静态IP地址
      subnetMask: string; //子网掩码
      staticDnsIp: string; //DNS服务器IP
    }) {
      return Request<null>("/guide/dhcp-client/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  ClientModel: {
    GET() {
      return Request<GuideClientModel>("/guide/client-mode/", {
        method: "GET",
      });
    },
    POST(data: GuideClientModel) {
      return Request<null>("/guide/client-mode/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  GatewayRouter: {
    //配置旁路由网络
    POST(data: GuideGateway) {
      return Request<null>("/guide/gateway-router/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  DockerStatus: {
    //查看docker状态
    GET() {
      return Request<GuideDockerStatus>("/guide/docker/status/", {
        method: "GET",
      });
    },
  },
  DockerPartitionList: {
    //docker推荐安装位置
    GET() {
      return Request<GuideDockerPartitionList>(
        "/guide/docker/partition/list/",
        {
          method: "GET",
        }
      );
    },
  },
  DockerTransfer: {
    //迁移docker
    POST(data: { path: string; force: boolean; overwriteDir: boolean }) {
      return Request<GuideDockerTransfer>("/guide/docker/transfer/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },

  DockerSwitch: {
    //docker开启或关闭
    POST(data: GuideDockerSwitchRequest) {
      return Request<null>("/guide/docker/switch/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },

  DownloadService: {
    //下载服务状态
    Status: {
      GET() {
        return Request<GuideDownloadServiceStatus>(
          "/guide/download-service/status/",
          {
            method: "GET",
          }
        );
      },
    },
  },
  DownloadPartition: {
    //下载目录推荐安装位置
    List: {
      GET() {
        return Request<GuideDownloadPartitionList>(
          "/guide/download/partition/list/",
          {
            method: "GET",
          }
        );
      },
    },
  },

  Aria2Init: {
    //配置aria2
    POST(data: {
      configPath: string;
      downloadPath: string;
      rpcToken: string;
      btTracker: string;
    }) {
      return Request<GuideAria2Init>("/guide/aria2/init/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  qbitorrentInit: {
    //配置qbitorrent
    POST(data: { configPath: string; downloadPath: string }) {
      return Request<GuideqBitorrentInit>("/guide/qbittorrent/init/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },

  transmissionInit: {
    //配置transmission
    POST(data: { configPath: string; downloadPath: string }) {
      return Request<GuideTransmissionInit>("/guide/transmission/init/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },

  GetLan: {
    //lan口网络信息
    GET() {
      return Request<GuideLanSetting>("/guide/lan/", {
        method: "GET",
      });
    },
  },
  LanIp: {
    //更换内网地址
    POST(data: GuideLanSetting) {
      return Request<null>("/guide/lan/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },

  SoftSource: {
    //配置软件源
    POST(data: { softSourceIdentity: string }) {
      return Request<GuideSoftSource>("/guide/soft-source/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },

  GetSoftSource: {
    //获取当前软件源配置
    GET() {
      return Request<GuideSoftSource>("/guide/soft-source/", {
        method: "GET",
      });
    },
  },

  SoftSourceList: {
    //获取软件源配置列表
    GET() {
      return Request<GuideSoftSourceList>("/guide/soft-source/list/", {
        method: "GET",
      });
    },
  },

  PostDdns: {
    POST(data: {
      ipVersion: string;
      serviceName: string;
      domain: string;
      userName: string;
      password: string;
    }) {
      return Request<null>("/u/guide/ddns/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  GetDdns: {
    GET() {
      return Request<GuideDdns>("/u/guide/ddns/", {
        method: "GET",
      });
    },
  },
  Ddnsto: {
    POST(data: { token: string }) {
      return Request<null>("/guide/ddnsto/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  DdntoConfig: {
    GET() {
      return Request<GuideDdntoConfig>("/guide/ddnsto/config/", {
        method: "GET",
      });
    },
  },
  DdnstoAddress: {
    POST(data: { address: string }) {
      return Request<null>("/guide/ddnsto/address/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
};

export const Raid = {
  //raid相关接口
  Create: {
    //创建raid
    POST(data: { level: string; devicePaths: string[] }) {
      return Request<null>("/raid/create/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  Delete: {
    //删除raid
    POST(data: { path: string; mountPath?: string; members: string[] }) {
      return Request<null>("/raid/delete/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  Add: {
    //扩充raid成员，新加入盘，并且直接投入使用
    POST(data: { path: string; memberPath: string }) {
      return Request<null>("/raid/add/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  Remove: {
    //删除raid成员
    POST(data: { path: string; memberPath: string }) {
      return Request<null>("/raid/remove/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  Recover: {
    //恢复raid成员，如果raid的成员盘满足最低数量，新增的盘则为预备盘
    POST(data: { path: string; memberPath: string }) {
      return Request<null>("/raid/recover/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  Detail: {
    //查看raid详细信息
    POST(data: { path: string }) {
      return Request<RaidDetail>("/raid/detail/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  List: {
    //获取raid列表数据
    GET() {
      return Request<RaidList>("/raid/list/", {
        method: "GET",
      });
    },
  },
  CreateList: {
    //创建raid,获取可用raid成员
    GET() {
      return Request<RaidCreateList>("/raid/create/list/", {
        method: "GET",
      });
    },
  },
  Autofix: {
    //扫描恢复raid
    POST() {
      return Request<unknown>("/raid/autofix/", {
        method: "POST",
      });
    },
  },
};

export const Smart = {
  Log: {
    GET() {
      return Request<ResponseSmartLog>("/smart/log/", {
        method: "GET",
      });
    },
  },
  List: {
    GET() {
      return Request<ResponseSmartList>("/u/smart/list/", {
        method: "GET",
      });
    },
  },
  Config: {
    GET() {
      return Request<ResponseSmartConfig>("/smart/config/", {
        method: "GET",
      });
    },
    POST(data: RequestSmartConfig) {
      return Request<ResponseSmartConfig>("/smart/config/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
  },
  Test: {
    POST(data: RequestSmartTest) {
      return Request<ResponseSmartTest>("/u/smart/test/", {
        method: "POST",
        headers: {
          "Content-Type": "application/json;charset=utf-8",
        },
        body: JSON.stringify(data),
      });
    },
    Result: {
      POST(data: RequestSmartTestResult) {
        return Request<ResponseSmartTestResult>("/smart/test/result/", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
          body: JSON.stringify(data),
        });
      },
    },
  },
  Attribute: {
    Result: {
      POST(data: RequestSmartAttributeResult) {
        return Request<ResponseSmartAttributeResult>(
          "/smart/attribute/result/",
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json;charset=utf-8",
            },
            body: JSON.stringify(data),
          }
        );
      },
    },
  },
  Extend: {
    Result: {
      POST(data: RequestSmartExtendResult) {
        return Request<ResponseSmartExtendResult>("/smart/extend/result/", {
          method: "POST",
          headers: {
            "Content-Type": "application/json;charset=utf-8",
          },
          body: JSON.stringify(data),
        });
      },
    },
  },
};

export const Quickwifi = {
  // 管理共享
  List: {
    GET() {
      return Request<ResponseIfaces>("/wireless/list-iface/", {
        method: "GET",
      });
    },
  },
  // 打开关闭wifi
  Switch: {
    POST(data: EnableIface) {
      return Request<any>("/wireless/enable-iface/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  // 功率修改
  Power: {
    POST(data: SetDevice) {
      return Request<any>("/wireless/set-device-power/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  // 编辑wifi
  Edit: {
    POST(data: Ifaces) {
      return Request<any>("/wireless/edit-iface/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  // 初始化wifi
  Setup: {
    POST(data: { [key: string]: Iface }) {
      return Request<unknown>("/wireless/setup/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
};

// 引导页需要的接口
export const GuidePage = {
  // 获取已安装的插件
  getInstalled: {
    GET() {
      return Request1<any>("/store/installed/", {
        method: "GET",
      });
    },
  },
  //检测是否需要向导和wifi模块
  needSetup: {
    GET() {
      return Request<GuideNeedSetupInfo>("/guide/need/setup/", {
        method: "GET",
      });
    },
  },
  // 设置系统密码
  setPassword: {
    POST(data: { [key: string]: string }) {
      return Request<unknown>("/system/setPassword/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  completeGuide: {
    POST() {
      return Request<unknown>("/guide/finish/setup/", {
        method: "POST",
      });
    },
  },
};

// 设备管理
export const DeviceMangement = {
  // 获取局域网内所有设备列表
  listDevices: {
    GET() {
      return Request<any>("/lanctrl/listDevices/", {
        method: "GET",
      });
    },
  },
  // 静态分配
  staticDeviceConfig: {
    POST(data: { [key: string]: any }) {
      return Request<unknown>("/lanctrl/staticDeviceConfig/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  //全局配置
  globalConfigs: {
    GET() {
      return Request<any>("/lanctrl/globalConfigs/", {
        method: "GET",
      });
    },
  },
  //限速配置
  speedLimitConfig: {
    POST(data: { [key: string]: any }) {
      return Request<unknown>("/lanctrl/speedLimitConfig/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  //静态分配的设备
  listStaticDevices: {
    GET() {
      return Request<any>("/lanctrl/listStaticDevices/", {
        method: "GET",
      });
    },
  },
  //限速设备
  listSpeedLimitedDevices: {
    GET() {
      return Request<any>("/lanctrl/listSpeedLimitedDevices/", {
        method: "GET",
      });
    },
  },
  //配置DHCP
  dhcpGatewayConfig: {
    POST(data: { [key: string]: any }) {
      return Request<any>("/lanctrl/dhcpGatewayConfig/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  dhcpTagsConfig: {
    POST(data: { [key: string]: any }) {
      return Request<any>("/lanctrl/dhcpTagsConfig/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  //   全局限速配置
  enableSpeedLimit: {
    POST(data: { [key: string]: any }) {
      return Request<any>("/lanctrl/enableSpeedLimit/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  //浮动网关地址
  enableFloatGateway: {
    POST(data: { [key: string]: any }) {
      return Request<any>("/lanctrl/enableFloatGateway/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
  // 获取设备速率
  speedsForDevices: {
    GET() {
      return Request<any>("/lanctrl/speedsForDevices/", {
        method: "GET",
      });
    },
  },
  //设备速率查询
  speedsForOneDevice: {
    POST(data: { [key: string]: any }) {
      return Request<any>("/lanctrl/speedsForOneDevice/", {
        body: JSON.stringify(data),
        method: "POST",
      });
    },
  },
};

// 设备管理
export const ModuleSettings = {
  GET() {
    return Request<any>("/system/module-settings/", {
      method: "GET",
    });
  },
  POST(data: { [key: string]: any }) {
    return Request<any>("/system/module-settings/", {
      body: JSON.stringify(data),
      method: "POST",
    });
  },
}
