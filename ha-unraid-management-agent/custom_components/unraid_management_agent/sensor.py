"""Sensor platform for Unraid Management Agent."""
from __future__ import annotations

import logging
from typing import Any

from homeassistant.components.sensor import (
    SensorDeviceClass,
    SensorEntity,
    SensorStateClass,
)
from homeassistant.config_entries import ConfigEntry
from homeassistant.const import (
    PERCENTAGE,
    UnitOfPower,
    UnitOfTemperature,
    UnitOfTime,
)
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddEntitiesCallback
from homeassistant.helpers.update_coordinator import CoordinatorEntity

from . import UnraidDataUpdateCoordinator
from .const import (
    ATTR_ARRAY_STATE,
    ATTR_CPU_CORES,
    ATTR_CPU_MODEL,
    ATTR_CPU_THREADS,
    ATTR_GPU_DRIVER_VERSION,
    ATTR_HOSTNAME,
    ATTR_NUM_DATA_DISKS,
    ATTR_NUM_DISKS,
    ATTR_NUM_PARITY_DISKS,
    ATTR_RAM_TOTAL,
    ATTR_SERVER_MODEL,
    DOMAIN,
    ICON_ARRAY,
    ICON_CPU,
    ICON_GPU,
    ICON_MEMORY,
    ICON_NETWORK,
    ICON_PARITY,
    ICON_POWER,
    ICON_TEMPERATURE,
    ICON_UPTIME,
    ICON_UPS,
    KEY_ARRAY,
    KEY_DISKS,
    KEY_GPU,
    KEY_NETWORK,
    KEY_SYSTEM,
    KEY_UPS,
    MANUFACTURER,
    MODEL,
)

_LOGGER = logging.getLogger(__name__)


async def async_setup_entry(
    hass: HomeAssistant,
    entry: ConfigEntry,
    async_add_entities: AddEntitiesCallback,
) -> None:
    """Set up Unraid sensor entities."""
    coordinator: UnraidDataUpdateCoordinator = hass.data[DOMAIN][entry.entry_id]

    entities: list[SensorEntity] = []

    # System sensors
    entities.extend([
        UnraidCPUUsageSensor(coordinator, entry),
        UnraidRAMUsageSensor(coordinator, entry),
        UnraidCPUTemperatureSensor(coordinator, entry),
        UnraidUptimeSensor(coordinator, entry),
    ])

    # Motherboard temperature sensor (if available)
    system_data = coordinator.data.get(KEY_SYSTEM, {})
    if system_data.get("motherboard_temp_celsius"):
        entities.append(UnraidMotherboardTemperatureSensor(coordinator, entry))

    # Fan sensors (dynamic, one per fan)
    fans = system_data.get("fans", [])
    for fan in fans:
        fan_name = fan.get("name", "unknown")
        entities.append(UnraidFanSensor(coordinator, entry, fan_name))

    # Array sensors
    entities.extend([
        UnraidArrayUsageSensor(coordinator, entry),
        UnraidParityProgressSensor(coordinator, entry),
    ])

    # Disk sensors (dynamic, one per disk)
    disks = coordinator.data.get(KEY_DISKS, [])
    for disk in disks:
        disk_id = disk.get("id", disk.get("name", "unknown"))
        disk_name = disk.get("name", disk_id)
        # Create sensors for each disk
        entities.extend([
            UnraidDiskUsageSensor(coordinator, entry, disk_id, disk_name),
            UnraidDiskTemperatureSensor(coordinator, entry, disk_id, disk_name),
        ])

    # GPU sensors (if GPU available)
    if coordinator.data.get(KEY_GPU):
        entities.extend([
            UnraidGPUNameSensor(coordinator, entry),
            UnraidGPUUtilizationSensor(coordinator, entry),
            UnraidGPUCPUTemperatureSensor(coordinator, entry),
            UnraidGPUPowerSensor(coordinator, entry),
        ])

    # UPS sensors (if UPS connected)
    if coordinator.data.get(KEY_UPS, {}).get("connected"):
        entities.extend([
            UnraidUPSBatterySensor(coordinator, entry),
            UnraidUPSLoadSensor(coordinator, entry),
            UnraidUPSRuntimeSensor(coordinator, entry),
        ])

    # Network sensors
    for interface in coordinator.data.get(KEY_NETWORK, []):
        interface_name = interface.get("name", "unknown")
        entities.extend([
            UnraidNetworkRXSensor(coordinator, entry, interface_name),
            UnraidNetworkTXSensor(coordinator, entry, interface_name),
        ])

    async_add_entities(entities)


class UnraidSensorBase(CoordinatorEntity, SensorEntity):
    """Base class for Unraid sensors."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator)
        self._attr_has_entity_name = True
        self._entry = entry

    @property
    def device_info(self) -> dict[str, Any]:
        """Return device information."""
        system_data = self.coordinator.data.get(KEY_SYSTEM, {})
        hostname = system_data.get("hostname", "Unraid")
        
        return {
            "identifiers": {(DOMAIN, self._entry.entry_id)},
            "name": f"Unraid ({hostname})",
            "manufacturer": MANUFACTURER,
            "model": MODEL,
            "sw_version": system_data.get("version", "Unknown"),
        }


# System Sensors

class UnraidCPUUsageSensor(UnraidSensorBase):
    """CPU usage sensor."""

    _attr_name = "CPU Usage"
    _attr_native_unit_of_measurement = PERCENTAGE
    _attr_device_class = SensorDeviceClass.POWER_FACTOR
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_CPU

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_cpu_usage"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_SYSTEM, {}).get("cpu_usage_percent")

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        system_data = self.coordinator.data.get(KEY_SYSTEM, {})
        return {
            ATTR_CPU_MODEL: system_data.get("cpu_model"),
            ATTR_CPU_CORES: system_data.get("cpu_cores"),
            ATTR_CPU_THREADS: system_data.get("cpu_threads"),
        }


class UnraidRAMUsageSensor(UnraidSensorBase):
    """RAM usage sensor."""

    _attr_name = "RAM Usage"
    _attr_native_unit_of_measurement = PERCENTAGE
    _attr_device_class = SensorDeviceClass.POWER_FACTOR
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_MEMORY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_ram_usage"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_SYSTEM, {}).get("ram_usage_percent")

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        system_data = self.coordinator.data.get(KEY_SYSTEM, {})
        ram_total = system_data.get("ram_total_bytes", 0)
        return {
            ATTR_RAM_TOTAL: f"{ram_total / (1024**3):.2f} GB" if ram_total else "Unknown",
            ATTR_SERVER_MODEL: system_data.get("server_model"),
        }


class UnraidCPUTemperatureSensor(UnraidSensorBase):
    """CPU temperature sensor."""

    _attr_name = "CPU Temperature"
    _attr_native_unit_of_measurement = UnitOfTemperature.CELSIUS
    _attr_device_class = SensorDeviceClass.TEMPERATURE
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_TEMPERATURE

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_cpu_temperature"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_SYSTEM, {}).get("cpu_temp_celsius")


class UnraidMotherboardTemperatureSensor(UnraidSensorBase):
    """Motherboard temperature sensor."""

    _attr_name = "Motherboard Temperature"
    _attr_native_unit_of_measurement = UnitOfTemperature.CELSIUS
    _attr_device_class = SensorDeviceClass.TEMPERATURE
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_TEMPERATURE

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_motherboard_temperature"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_SYSTEM, {}).get("motherboard_temp_celsius")


class UnraidFanSensor(UnraidSensorBase):
    """Fan speed sensor."""

    _attr_native_unit_of_measurement = "RPM"
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = "mdi:fan"

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        fan_name: str,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._fan_name = fan_name
        self._attr_name = f"Fan {fan_name}"

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        # Sanitize fan name for unique ID
        safe_name = self._fan_name.replace(" ", "_").replace("/", "_").lower()
        return f"{self._entry.entry_id}_fan_{safe_name}"

    @property
    def native_value(self) -> int | None:
        """Return the state."""
        fans = self.coordinator.data.get(KEY_SYSTEM, {}).get("fans", [])
        for fan in fans:
            if fan.get("name") == self._fan_name:
                return fan.get("rpm")
        return None


class UnraidUptimeSensor(UnraidSensorBase):
    """Uptime sensor."""

    _attr_name = "Uptime"
    _attr_native_unit_of_measurement = UnitOfTime.SECONDS
    _attr_device_class = SensorDeviceClass.DURATION
    _attr_state_class = SensorStateClass.TOTAL_INCREASING
    _attr_icon = ICON_UPTIME

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_uptime"

    @property
    def native_value(self) -> int | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_SYSTEM, {}).get("uptime_seconds")

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        system_data = self.coordinator.data.get(KEY_SYSTEM, {})
        return {
            ATTR_HOSTNAME: system_data.get("hostname"),
        }


# Array Sensors

class UnraidArrayUsageSensor(UnraidSensorBase):
    """Array usage sensor."""

    _attr_name = "Array Usage"
    _attr_native_unit_of_measurement = PERCENTAGE
    _attr_device_class = SensorDeviceClass.POWER_FACTOR
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_ARRAY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_array_usage"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_ARRAY, {}).get("used_percent")

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        array_data = self.coordinator.data.get(KEY_ARRAY, {})
        return {
            ATTR_ARRAY_STATE: array_data.get("state"),
            ATTR_NUM_DISKS: array_data.get("num_disks"),
            ATTR_NUM_DATA_DISKS: array_data.get("num_data_disks"),
            ATTR_NUM_PARITY_DISKS: array_data.get("num_parity_disks"),
        }


class UnraidParityProgressSensor(UnraidSensorBase):
    """Parity check progress sensor."""

    _attr_name = "Parity Check Progress"
    _attr_native_unit_of_measurement = PERCENTAGE
    _attr_device_class = SensorDeviceClass.POWER_FACTOR
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_PARITY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_parity_progress"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_ARRAY, {}).get("parity_check_progress")


# GPU Sensors

class UnraidGPUNameSensor(UnraidSensorBase):
    """GPU name sensor."""

    _attr_name = "GPU Name"
    _attr_icon = ICON_GPU

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_gpu_name"

    @property
    def native_value(self) -> str | None:
        """Return the state."""
        gpu_list = self.coordinator.data.get(KEY_GPU, [])
        if gpu_list and len(gpu_list) > 0:
            return gpu_list[0].get("name")
        return None

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        gpu_list = self.coordinator.data.get(KEY_GPU, [])
        if gpu_list and len(gpu_list) > 0:
            return {
                ATTR_GPU_DRIVER_VERSION: gpu_list[0].get("driver_version"),
            }
        return {}


class UnraidGPUUtilizationSensor(UnraidSensorBase):
    """GPU utilization sensor."""

    _attr_name = "GPU Utilization"
    _attr_native_unit_of_measurement = PERCENTAGE
    _attr_device_class = SensorDeviceClass.POWER_FACTOR
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_GPU

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_gpu_utilization"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        gpu_list = self.coordinator.data.get(KEY_GPU, [])
        if gpu_list and len(gpu_list) > 0:
            return gpu_list[0].get("utilization_gpu_percent")
        return None


class UnraidGPUCPUTemperatureSensor(UnraidSensorBase):
    """GPU CPU temperature sensor (for iGPUs)."""

    _attr_name = "GPU CPU Temperature"
    _attr_native_unit_of_measurement = UnitOfTemperature.CELSIUS
    _attr_device_class = SensorDeviceClass.TEMPERATURE
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_TEMPERATURE

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_gpu_cpu_temperature"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        gpu_list = self.coordinator.data.get(KEY_GPU, [])
        if gpu_list and len(gpu_list) > 0:
            return gpu_list[0].get("cpu_temperature_celsius")
        return None


class UnraidGPUPowerSensor(UnraidSensorBase):
    """GPU power consumption sensor."""

    _attr_name = "GPU Power"
    _attr_native_unit_of_measurement = UnitOfPower.WATT
    _attr_device_class = SensorDeviceClass.POWER
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_POWER

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_gpu_power"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        gpu_list = self.coordinator.data.get(KEY_GPU, [])
        if gpu_list and len(gpu_list) > 0:
            return gpu_list[0].get("power_draw_watts")
        return None


# UPS Sensors

class UnraidUPSBatterySensor(UnraidSensorBase):
    """UPS battery sensor."""

    _attr_name = "UPS Battery"
    _attr_native_unit_of_measurement = PERCENTAGE
    _attr_device_class = SensorDeviceClass.BATTERY
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_UPS

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_ups_battery"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_UPS, {}).get("battery_charge_percent")


class UnraidUPSLoadSensor(UnraidSensorBase):
    """UPS load sensor."""

    _attr_name = "UPS Load"
    _attr_native_unit_of_measurement = PERCENTAGE
    _attr_device_class = SensorDeviceClass.POWER_FACTOR
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_UPS

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_ups_load"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_UPS, {}).get("load_percent")


class UnraidUPSRuntimeSensor(UnraidSensorBase):
    """UPS runtime sensor."""

    _attr_name = "UPS Runtime"
    _attr_native_unit_of_measurement = UnitOfTime.SECONDS
    _attr_device_class = SensorDeviceClass.DURATION
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_UPS

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_ups_runtime"

    @property
    def native_value(self) -> int | None:
        """Return the state."""
        return self.coordinator.data.get(KEY_UPS, {}).get("runtime_seconds")


# Network Sensors

class UnraidNetworkRXSensor(UnraidSensorBase):
    """Network RX sensor."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        interface_name: str,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._interface_name = interface_name
        self._attr_name = f"Network {interface_name} RX"
        self._attr_icon = ICON_NETWORK

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_network_{self._interface_name}_rx"

    @property
    def native_value(self) -> int | None:
        """Return the state."""
        for interface in self.coordinator.data.get(KEY_NETWORK, []):
            if interface.get("name") == self._interface_name:
                return interface.get("bytes_received")
        return None


class UnraidNetworkTXSensor(UnraidSensorBase):
    """Network TX sensor."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        interface_name: str,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._interface_name = interface_name
        self._attr_name = f"Network {interface_name} TX"
        self._attr_icon = ICON_NETWORK

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_network_{self._interface_name}_tx"

    @property
    def native_value(self) -> int | None:
        """Return the state."""
        for interface in self.coordinator.data.get(KEY_NETWORK, []):
            if interface.get("name") == self._interface_name:
                return interface.get("bytes_sent")
        return None


class UnraidDiskUsageSensor(UnraidSensorBase):
    """Disk usage sensor."""

    _attr_native_unit_of_measurement = PERCENTAGE
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = "mdi:harddisk"

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        disk_id: str,
        disk_name: str,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._disk_id = disk_id
        self._disk_name = disk_name
        self._attr_name = f"Disk {disk_name} Usage"

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        # Sanitize disk ID for unique ID
        safe_id = self._disk_id.replace(" ", "_").replace("/", "_").lower()
        return f"{self._entry.entry_id}_disk_{safe_id}_usage"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        for disk in self.coordinator.data.get(KEY_DISKS, []):
            disk_id = disk.get("id", disk.get("name"))
            if disk_id == self._disk_id:
                return disk.get("usage_percent")
        return None

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        for disk in self.coordinator.data.get(KEY_DISKS, []):
            disk_id = disk.get("id", disk.get("name"))
            if disk_id == self._disk_id:
                size_bytes = disk.get("size_bytes", 0)
                used_bytes = disk.get("used_bytes", 0)
                free_bytes = disk.get("free_bytes", 0)
                return {
                    "device": disk.get("device"),
                    "status": disk.get("status"),
                    "filesystem": disk.get("filesystem"),
                    "mount_point": disk.get("mount_point"),
                    "size": f"{size_bytes / (1024**3):.2f} GB" if size_bytes else "Unknown",
                    "used": f"{used_bytes / (1024**3):.2f} GB" if used_bytes else "Unknown",
                    "free": f"{free_bytes / (1024**3):.2f} GB" if free_bytes else "Unknown",
                    "smart_status": disk.get("smart_status"),
                    "smart_errors": disk.get("smart_errors", 0),
                }
        return {}


class UnraidDiskTemperatureSensor(UnraidSensorBase):
    """Disk temperature sensor."""

    _attr_native_unit_of_measurement = UnitOfTemperature.CELSIUS
    _attr_device_class = SensorDeviceClass.TEMPERATURE
    _attr_state_class = SensorStateClass.MEASUREMENT
    _attr_icon = ICON_TEMPERATURE

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        disk_id: str,
        disk_name: str,
    ) -> None:
        """Initialize the sensor."""
        super().__init__(coordinator, entry)
        self._disk_id = disk_id
        self._disk_name = disk_name
        self._attr_name = f"Disk {disk_name} Temperature"

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        # Sanitize disk ID for unique ID
        safe_id = self._disk_id.replace(" ", "_").replace("/", "_").lower()
        return f"{self._entry.entry_id}_disk_{safe_id}_temperature"

    @property
    def native_value(self) -> float | None:
        """Return the state."""
        for disk in self.coordinator.data.get(KEY_DISKS, []):
            disk_id = disk.get("id", disk.get("name"))
            if disk_id == self._disk_id:
                temp = disk.get("temperature_celsius")
                # Return None if disk is spun down (temp = 0 or missing)
                return temp if temp and temp > 0 else None
        return None

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        for disk in self.coordinator.data.get(KEY_DISKS, []):
            disk_id = disk.get("id", disk.get("name"))
            if disk_id == self._disk_id:
                return {
                    "device": disk.get("device"),
                    "status": disk.get("status"),
                    "power_on_hours": disk.get("power_on_hours"),
                    "power_cycle_count": disk.get("power_cycle_count"),
                }
        return {}

