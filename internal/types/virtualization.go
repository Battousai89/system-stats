package types

import (
	"fmt"

	"system-stats/internal/helpers"
)

// VirtualizationInfo информация о виртуализации
type VirtualizationInfo struct {
	Virtualized    bool   `json:"virtualized"`     // Является ли ВМ
	Hypervisor     string `json:"hypervisor"`      // Тип гипервизора
	GuestType      string `json:"guest_type"`      // Тип гостевой ОС
	ContainerType  string `json:"container_type"`  // Тип контейнера (docker, lxc, etc.)
	Architecture   string `json:"architecture"`    // Архитектура
	BootTime       uint64 `json:"boot_time"`       // Время загрузки (unix timestamp)
	Platform       string `json:"platform"`        // Платформа
	PlatformFamily string `json:"platform_family"` // Семейство платформы
	PlatformVersion string `json:"platform_version"` // Версия платформы
}

// ToPrint форматирует VirtualizationInfo для вывода
func (v *VirtualizationInfo) ToPrint() string {
	b := helpers.NewBuilder()

	if v.Virtualized {
		b.AddField("Virtualized", "Yes", "")
		b.AddField("Hypervisor", v.Hypervisor, "")
	} else {
		b.AddField("Virtualized", "No", "")
	}

	if v.GuestType != "" {
		b.AddField("Guest Type", v.GuestType, "")
	}
	if v.ContainerType != "" {
		b.AddField("Container", v.ContainerType, "")
	}

	b.AddField("Platform", v.Platform, "")
	b.AddField("Platform Family", v.PlatformFamily, "")
	b.AddField("Platform Version", v.PlatformVersion, "")
	b.AddField("Architecture", v.Architecture, "")

	return b.Build()
}

// NewVirtualizationInfo создает новую VirtualizationInfo
func NewVirtualizationInfo() *VirtualizationInfo {
	return &VirtualizationInfo{}
}

// String возвращает строковое представление
func (v *VirtualizationInfo) String() string {
	if v.Virtualized {
		return fmt.Sprintf("Virtualized (%s)", v.Hypervisor)
	}
	return "Bare Metal"
}
