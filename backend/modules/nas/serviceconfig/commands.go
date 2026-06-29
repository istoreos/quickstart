package serviceconfig

import "fmt"

func BuildSambaShareCommands(existingShareCount int, input SambaCreateInput) []string {
	target := fmt.Sprintf("@sambashare[%v]", existingShareCount)

	cmdList := []string{
		"uci add samba4 sambashare",
		"uci set samba4.@samba[0].enabled=1",
		"uci set samba4.@samba[0].macos=1",
		fmt.Sprintf("uci set samba4.%v.name=%v", target, input.ShareName),
		fmt.Sprintf("uci set samba4.%v.path=%v", target, input.RootPath),
		fmt.Sprintf("uci set samba4.%v.read_only=%v", target, "no"),
		fmt.Sprintf("uci set samba4.%v.users=%v", target, input.Username),
		fmt.Sprintf("uci set samba4.%v.create_mask=%v", target, "0777"),
		fmt.Sprintf("uci set samba4.%v.force_root=%v", target, "1"),
	}
	if input.AllowLegacy {
		cmdList = append(cmdList, "uci set samba4.@samba[0].allow_legacy_protocols=1")
	}
	return append(cmdList, "uci commit samba4", "/etc/init.d/samba4 restart")
}

func BuildWebdavConfigCommands(input WebdavCreateInput) []string {
	return []string{
		fmt.Sprintf("uci set gowebdav.config.root_dir=%v", input.RootPath),
		"uci set gowebdav.config.enable=1",
		fmt.Sprintf("uci set gowebdav.config.username=%v", input.Username),
		fmt.Sprintf("uci set gowebdav.config.password=%v", input.Password),
		"uci set gowebdav.config.allow_wan=1",
		"uci commit gowebdav",
	}
}
