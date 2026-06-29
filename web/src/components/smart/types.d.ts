declare type ActionSmartEditDiskPropsNext = (device: SmartConfigDevice) => Promise<void>
declare type ActionSmartEditDiskPropsDevice = SmartConfigDevice | null

declare interface ActionSmartEditDiskProps {
    disk: SmartDiskInfo,
    device?: ActionSmartEditDiskPropsDevice
    next: ActionSmartEditDiskPropsNext
}
declare type ActionSmartEditDisk = (props: ActionSmartEditDiskProps) => void


declare type ActionSmartAddTaskPropsNext = (task: SmartConfigTask) => Promise<void>
declare interface ActionSmartAddTaskProps {
    disks: SmartDisks
    config: PropsSmartConfig
    next: ActionSmartAddTaskPropsNext
}
declare type ActionSmartAddTask = (props: ActionSmartAddTaskProps) => void



declare type ActionSmartTestTask = (props: {
    task: SmartConfigTask
}) => void


declare type ActionSmartDiskInfo = (props: {
    disk: SmartDiskInfo
}) => void