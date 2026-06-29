declare type RaidSetupType = "create" | "info" | "edit" | "remove" | "recover"
declare type ActionRaidSuccess = () => void
declare interface ActionRaidProps {
    setup: RaidSetupType
    raid?: Disksinfo
    success?: ActionRaidSuccess
}
declare type ActionRaid = (props: ActionRaidProps) => {
    Close: () => void
}