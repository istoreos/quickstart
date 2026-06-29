package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/linkease/quick-start/istore-backend/service"
	"github.com/linkease/quick-start/istore-backend/utils"
	"github.com/manifoldco/promptui"
)

type mainSelection int

const (
	selectIPShow mainSelection = iota
	selectIPModify
	selectWanAccess
	selectInstallX86
	selectReset
)

const (
	selectQuit mainSelection = 255
)

var (
	mainMenu = map[string]mainSelection{
		"0、Show Interfaces":  selectIPShow,
		"1、Change LAN IP":    selectIPModify,
		"2、Allow WAN Access": selectWanAccess,
		"3、Install X86":      selectInstallX86,
		"4、Reset":            selectReset,
		"q、QUIT":             selectQuit,
	}
)

// https://github.com/nicksnyder/go-i18n
func getMainSelection(mainItems []string) (mainSelection, error) {
	prompt := promptui.Select{
		Label: "Console",
		Items: mainItems,
	}

	_, result, err := prompt.Run()
	if err != nil {
		os.Exit(-1)
		return mainSelection(selectQuit), nil
	}
	if idx, ok := mainMenu[result]; ok {
		return mainSelection(idx), nil
	}

	return -2, errors.New("item not found")
}

func mainPrompt() error {
	mainItems := make([]string, len(mainMenu))
	i := 0
	for k, v := range mainMenu {
		if v == selectInstallX86 && !(runtime.GOARCH == "amd64" && runtime.GOOS == "linux") {
			continue
		}
		if v == selectWanAccess && !service.IsWanPresent() {
			continue
		}
		mainItems[i] = k
		i++
	}
	mainItems = mainItems[:i]
	sort.Strings(mainItems)

MAINLOOP:
	for {
		selectIdx, err := getMainSelection(mainItems)
		if err != nil {
			return err
		}
		switch selectIdx {
		case selectIPModify:
			promptIpModify()
			continue MAINLOOP
		case selectIPShow:
			err := promptIpShow()
			if err != nil {
				fmt.Println("error:", err)
				return nil
			}
			continue MAINLOOP
		case selectWanAccess:
			promptWanAccess()
			continue MAINLOOP
		case selectReset:
			promptReset()
			return nil
		case selectInstallX86:
			err := promptInstallX86()
			if err != nil {
				fmt.Println("error:", err)
				return nil
			}
			continue MAINLOOP
		case selectQuit:
			break MAINLOOP
		}
	}

	return nil
}

func promptIp() (string, error) {
	ipValidate :=
		func(input string) error {
			ip := net.ParseIP(input)
			if ip == nil || ip.To4() == nil {
				return errors.New("IP Invalid")
			}
			return nil
		}

	ipPrompt := promptui.Prompt{
		Label:    "IP",
		Validate: ipValidate,
	}

	return ipPrompt.Run()
}

func promptMask() (string, error) {
	maskValidate :=
		func(input string) error {
			ip := net.ParseIP(input)
			if ip == nil || ip.To4() == nil {
				return errors.New("MASK Invalid")
			}
			if !utils.IsValidIpv4Mask(ip) {
				return errors.New("MASK Invalid")
			}
			return nil
		}

	maskPrompt := promptui.Prompt{
		Label:    "MASK(eg. 255.255.255.0)",
		Validate: maskValidate,
	}

	return maskPrompt.Run()
}

func promptIpModify() error {
	ip, err := promptIp()
	if err != nil {
		return err
	}

	mask, err := promptMask()
	if err != nil {
		return err
	}

	err = service.LanSetting(ip, mask, false)
	if err == nil {
		fmt.Println("OK", "ip=", ip, "mask=", mask)
	} else {
		fmt.Println("FAILED", "ip=", ip, "mask=", mask)
	}
	return nil
}

func promptIpShow() error {
	ctx := context.Background()
	interfaceResp, err := service.NetworkInterfaceStatus(ctx)
	if err != nil {
		return err
	}
	intrs := interfaceResp.Result.Interfaces
	if len(intrs) == 0 {
		fmt.Println("No interfaces found!")
		return errors.New("No interfaces found")
	}
	l := list.NewWriter()
	l.SetStyle(list.StyleBulletCircle)
	for _, v := range intrs {
		l.AppendItem(strings.ToUpper(v.Name))
		l.Indent()
		l.AppendItem("IPv4: " + v.IPV4Addr)
		l.AppendItem("IPv6: " + v.IPV6Addr)
		l.AppendItem("DeviceName: " + v.PortName)
		l.AppendItem("Devices:")
		l.Indent()
		for _, port := range v.Ports {
			l.AppendItem("Name: " + port.Name)
			l.AppendItem("Link: " + port.LinkState)
			l.AppendItem("Speed: " + port.LinkSpeed)
		}
		l.UnIndent()
		l.UnIndent()
	}
	printIpHeader("Interfaces", l.Render(), "")

	return nil
}

func promptWanAccess() error {
	prompt := promptui.Select{
		Label: "Allow access this device from WAN port, may cause security risks",
		Items: []string{
			"OK, Continue",
			"NO",
		},
	}
	_, result, err := prompt.Run()
	if err != nil {
		return err
	}
	if result == "NO" {
		return nil
	}
	service.WanAccessAllow(context.Background())
	fmt.Println("OK")
	return nil
}

func printIpHeader(title string, content string, prefix string) {
	fmt.Printf("%s:\n", title)
	fmt.Println(strings.Repeat("-", len(title)+1))
	for _, line := range strings.Split(content, "\n") {
		fmt.Printf("%s%s\n", prefix, line)
	}
	fmt.Println()
}

type verboseReader struct {
	io.Reader
	total    int64
	step     int64
	nextStep int64
}

func (r *verboseReader) Read(buf []byte) (n int, err error) {
	n, err = r.Reader.Read(buf)
	if err == nil {
		r.total += int64(n)
		if r.total > r.nextStep {
			r.nextStep += r.step
			fmt.Print(".")
		}
	} else if err == io.EOF {
		fmt.Println("\nRead OK, Writing, Please wait!")
	}
	return n, err
}

func promptInstallX86() error {
	x86Install, err := service.GetDiskInfoForX86Install()
	if err != nil {
		return err
	}
	if len(x86Install.Devs) == 0 {
		fmt.Println("Warning: No disk to install, please check! (Only support disk > 2.5 GiB)")
		return errors.New("No disk to install")
	}

	devItems := make([]string, 0, len(x86Install.Devs)+1)
	for _, v := range x86Install.Devs {
		devItems = append(devItems, v.Target+" [ "+v.SizeStr+" ] "+v.DisplayName)
	}
	devItems = append(devItems, "QUIT")

	for {
		prompt := promptui.Select{
			Label: "Select Disk to install (Only show disk > 2.5 GiB)",
			Items: devItems,
		}

		index, result, err := prompt.Run()
		if err != nil {
			return err
		}
		if index >= len(x86Install.Devs) {
			break
		}

		dev := x86Install.Devs[index]
		prompt = promptui.Select{
			Label: fmt.Sprintf("Install %s to %s, Will Loss All Data", x86Install.Root.Name, dev.Target),
			Items: []string{
				"OK, Continue",
				"NO",
			},
		}
		_, result, err = prompt.Run()
		if err != nil {
			return err
		}
		if result == "NO" {
			continue
		}

		prompt = promptui.Select{
			Label: fmt.Sprintf("Warning Again!!! Loss All %s Data", dev.Target),
			Items: []string{
				"Install And Reboot",
				"NO",
				"QUIT",
			},
		}
		_, result, err = prompt.Run()
		if err != nil {
			return err
		}
		if result == "NO" {
			continue
		} else if result == "QUIT" {
			break
		}

		if len(dev.Mountpoints) > 0 {
			umounts := make([]string, 0, len(dev.Mountpoints))
			for _, v := range dev.Mountpoints {
				umounts = append(umounts, "umount '"+v+"'")
			}
			utils.BatchRun(context.TODO(), umounts, 10)
		}

		sourceDev := "/dev/" + x86Install.Root.Name
		reader, err := os.Open(sourceDev)
		if err != nil {
			return err
		}
		defer reader.Close()
		vr := &verboseReader{
			Reader: reader,
			step:   x86Install.Root.End / 20,
		}
		vr.nextStep = vr.step
		err = utils.Dd(vr, dev.Target, 4096, x86Install.Root.End)
		if err != nil {
			return err
		}
		target1, err := os.OpenFile(dev.Target, os.O_RDWR, 0)
		if err != nil {
			return err
		}
		defer target1.Close()
		target1.WriteAt([]byte("RESET000"), x86Install.Root.End)
		target1.Close()

		if x86Install.Root.PType == "GPT" {
			// fix GPT
			utils.BatchRun(context.TODO(), []string{
				"from=" + x86Install.Root.Name,
				"to=" + dev.Name,
				"boot_parts=3",
				// rebuild target disk GPT
				`parted -s "/dev/$to" mktable gpt`,
				// copy mbr disk id
				`dd if="/dev/$from" of="/dev/$to" bs=1 skip=440 seek=440 count=4 conv=notrunc,fsync 2>/dev/null`,
				// copy gpt disk guid
				`dd if="/dev/$from" of="/dev/$to" bs=8 skip=71 seek=71 count=2 conv=notrunc,fsync 2>/dev/null`,
				// copy gpt partition entries
				`dd if="/dev/$from" of="/dev/$to" bs=128 skip=8 seek=8 count=$boot_parts conv=notrunc,fsync 2>/dev/null`,
				// copy bios_grub partition entry
				`dd if="/dev/$from" of="/dev/$to" bs=128 skip=135 seek=135 count=1 conv=notrunc,fsync 2>/dev/null`,
			}, 0)

			crc32Ctx := crc32.NewIEEE()
			target, err := os.OpenFile(dev.Target, os.O_RDWR, 0)
			if err != nil {
				return err
			}
			defer target.Close()

			// fix GPT CRC32 of partition entries (little endian)
			target.Seek(1024, os.SEEK_SET)
			io.CopyN(crc32Ctx, target, 16384)
			crc32result := crc32Ctx.Sum32()

			uint32b := make([]byte, 4)
			binary.LittleEndian.PutUint32(uint32b, crc32result)
			target.WriteAt(uint32b, 600)

			// fix GPT CRC32 of header
			crc32Ctx.Reset()
			target.Seek(512, os.SEEK_SET)
			io.CopyN(crc32Ctx, target, 16)
			binary.LittleEndian.PutUint32(uint32b, 0)
			crc32Ctx.Write(uint32b)
			target.Seek(532, os.SEEK_SET)
			io.CopyN(crc32Ctx, target, 72)
			crc32result = crc32Ctx.Sum32()
			binary.LittleEndian.PutUint32(uint32b, crc32result)
			target.WriteAt(uint32b, 528)
			target.Close()

			// fix alternative GPT
			utils.BatchRun(context.TODO(), []string{
				"to=" + dev.Name,
				"boot_parts=3",
				`echo Fix | parted "/dev/$to" print >/dev/null`,
				"for offset in `seq 2 $boot_parts` ; do",
				`    parted -s "/dev/$to" set $offset msftdata on`,
				"done",
				`partprobe "/dev/$to"`,
			}, 0)
		}

		utils.BatchRun(context.TODO(), []string{
			"sync",
			"echo 3 >/proc/sys/vm/drop_caches",
		}, 0)

		// reboot
		fmt.Println("Please Remove the USB manually, wait 3s to reboot...")
		time.Sleep(time.Second * 3)
		fmt.Println("Rebooting")
		utils.BatchRun(context.Background(), []string{"reboot now > /dev/null 2>&1"}, 0)
		os.Exit(0)
	}

	return nil
}

func promptReset() error {
	prompt := promptui.Select{
		Label: "Reset to factory? will loss all configs",
		Items: []string{
			"YES",
			"NO",
		},
	}
	_, result, err := prompt.Run()
	if err != nil {
		return err
	}
	if result == "YES" {
		fmt.Println("Reseting and reboot...")
		utils.BatchRun(context.Background(), []string{"firstboot -y -r > /dev/null 2>&1"}, 0)
	}
	return nil
}
