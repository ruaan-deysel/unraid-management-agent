"""Binary sensor platform for Unraid Management Agent."""
from __future__ import annotations

import logging
from typing import Any

from homeassistant.components.binary_sensor import (
    BinarySensorDeviceClass,
    BinarySensorEntity,
)
from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddEntitiesCallback
from homeassistant.helpers.update_coordinator import CoordinatorEntity

from . import UnraidDataUpdateCoordinator
from .const import (
    ATTR_CONTAINER_IMAGE,
    ATTR_CONTAINER_PORTS,
    ATTR_PARITY_CHECK_STATUS,
    ATTR_VM_VCPUS,
    DOMAIN,
    ICON_ARRAY,
    ICON_CONTAINER,
    ICON_NETWORK,
    ICON_PARITY,
    ICON_UPS,
    ICON_VM,
    KEY_ARRAY,
    KEY_CONTAINERS,
    KEY_NETWORK,
    KEY_SYSTEM,
    KEY_UPS,
    KEY_VMS,
    MANUFACTURER,
    MODEL,
)

_LOGGER = logging.getLogger(__name__)


async def async_setup_entry(
    hass: HomeAssistant,
    entry: ConfigEntry,
    async_add_entities: AddEntitiesCallback,
) -> None:
    """Set up Unraid binary sensor entities."""
    coordinator: UnraidDataUpdateCoordinator = hass.data[DOMAIN][entry.entry_id]

    entities: list[BinarySensorEntity] = []

    # Array binary sensors
    entities.extend([
        UnraidArrayStartedBinarySensor(coordinator, entry),
        UnraidParityCheckRunningBinarySensor(coordinator, entry),
        UnraidParityValidBinarySensor(coordinator, entry),
    ])

    # UPS binary sensor (if UPS exists)
    if coordinator.data.get(KEY_UPS):
        entities.append(UnraidUPSConnectedBinarySensor(coordinator, entry))

    # Container binary sensors
    for container in coordinator.data.get(KEY_CONTAINERS, []):
        container_id = container.get("id") or container.get("container_id")
        container_name = container.get("name", "unknown")
        if container_id:
            entities.append(
                UnraidContainerBinarySensor(coordinator, entry, container_id, container_name)
            )

    # VM binary sensors
    for vm in coordinator.data.get(KEY_VMS, []):
        vm_id = vm.get("id") or vm.get("name")
        vm_name = vm.get("name", "unknown")
        if vm_id:
            entities.append(UnraidVMBinarySensor(coordinator, entry, vm_id, vm_name))

    # Network interface binary sensors
    for interface in coordinator.data.get(KEY_NETWORK, []):
        interface_name = interface.get("name", "unknown")
        entities.append(
            UnraidNetworkInterfaceBinarySensor(coordinator, entry, interface_name)
        )

    async_add_entities(entities)


class UnraidBinarySensorBase(CoordinatorEntity, BinarySensorEntity):
    """Base class for Unraid binary sensors."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
    ) -> None:
        """Initialize the binary sensor."""
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


# Array Binary Sensors

class UnraidArrayStartedBinarySensor(UnraidBinarySensorBase):
    """Array started binary sensor."""

    _attr_name = "Array Started"
    _attr_device_class = BinarySensorDeviceClass.RUNNING
    _attr_icon = ICON_ARRAY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_array_started"

    @property
    def is_on(self) -> bool:
        """Return true if array is started."""
        state = self.coordinator.data.get(KEY_ARRAY, {}).get("state", "").lower()
        return state == "started"


class UnraidParityCheckRunningBinarySensor(UnraidBinarySensorBase):
    """Parity check running binary sensor."""

    _attr_name = "Parity Check Running"
    _attr_device_class = BinarySensorDeviceClass.RUNNING
    _attr_icon = ICON_PARITY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_parity_check_running"

    @property
    def is_on(self) -> bool:
        """Return true if parity check is running."""
        status = self.coordinator.data.get(KEY_ARRAY, {}).get("parity_check_status", "").lower()
        return status == "running"

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        array_data = self.coordinator.data.get(KEY_ARRAY, {})
        return {
            ATTR_PARITY_CHECK_STATUS: array_data.get("parity_check_status"),
        }


class UnraidParityValidBinarySensor(UnraidBinarySensorBase):
    """Parity valid binary sensor."""

    _attr_name = "Parity Valid"
    _attr_device_class = BinarySensorDeviceClass.PROBLEM
    _attr_icon = ICON_PARITY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_parity_valid"

    @property
    def is_on(self) -> bool:
        """Return true if parity is valid (inverted for problem device class)."""
        # For PROBLEM device class, ON means there IS a problem
        # So we invert: parity_valid=true means NO problem (OFF)
        parity_valid = self.coordinator.data.get(KEY_ARRAY, {}).get("parity_valid", True)
        return not parity_valid


# UPS Binary Sensor

class UnraidUPSConnectedBinarySensor(UnraidBinarySensorBase):
    """UPS connected binary sensor."""

    _attr_name = "UPS Connected"
    _attr_device_class = BinarySensorDeviceClass.CONNECTIVITY
    _attr_icon = ICON_UPS

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_ups_connected"

    @property
    def is_on(self) -> bool:
        """Return true if UPS is connected."""
        return self.coordinator.data.get(KEY_UPS, {}).get("connected", False)


# Container Binary Sensors

class UnraidContainerBinarySensor(UnraidBinarySensorBase):
    """Container running binary sensor."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        container_id: str,
        container_name: str,
    ) -> None:
        """Initialize the binary sensor."""
        super().__init__(coordinator, entry)
        self._container_id = container_id
        self._container_name = container_name
        self._attr_name = f"Container {container_name}"
        self._attr_device_class = BinarySensorDeviceClass.RUNNING
        self._attr_icon = ICON_CONTAINER

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_container_{self._container_id}"

    @property
    def is_on(self) -> bool:
        """Return true if container is running."""
        for container in self.coordinator.data.get(KEY_CONTAINERS, []):
            cid = container.get("id") or container.get("container_id")
            if cid == self._container_id:
                state = container.get("state", "").lower()
                return state == "running"
        return False

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        for container in self.coordinator.data.get(KEY_CONTAINERS, []):
            cid = container.get("id") or container.get("container_id")
            if cid == self._container_id:
                return {
                    ATTR_CONTAINER_IMAGE: container.get("image"),
                    ATTR_CONTAINER_PORTS: container.get("ports"),
                }
        return {}


# VM Binary Sensors

class UnraidVMBinarySensor(UnraidBinarySensorBase):
    """VM running binary sensor."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        vm_id: str,
        vm_name: str,
    ) -> None:
        """Initialize the binary sensor."""
        super().__init__(coordinator, entry)
        self._vm_id = vm_id
        self._vm_name = vm_name
        self._attr_name = f"VM {vm_name}"
        self._attr_device_class = BinarySensorDeviceClass.RUNNING
        self._attr_icon = ICON_VM

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_vm_{self._vm_id}"

    @property
    def is_on(self) -> bool:
        """Return true if VM is running."""
        for vm in self.coordinator.data.get(KEY_VMS, []):
            vid = vm.get("id") or vm.get("name")
            if vid == self._vm_id:
                state = vm.get("state", "").lower()
                return state == "running"
        return False

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Return extra attributes."""
        for vm in self.coordinator.data.get(KEY_VMS, []):
            vid = vm.get("id") or vm.get("name")
            if vid == self._vm_id:
                return {
                    ATTR_VM_VCPUS: vm.get("vcpus"),
                }
        return {}


# Network Interface Binary Sensors

class UnraidNetworkInterfaceBinarySensor(UnraidBinarySensorBase):
    """Network interface up/down binary sensor."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        interface_name: str,
    ) -> None:
        """Initialize the binary sensor."""
        super().__init__(coordinator, entry)
        self._interface_name = interface_name
        self._attr_name = f"Network {interface_name}"
        self._attr_device_class = BinarySensorDeviceClass.CONNECTIVITY
        self._attr_icon = ICON_NETWORK

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_network_{self._interface_name}"

    @property
    def is_on(self) -> bool:
        """Return true if interface is up."""
        for interface in self.coordinator.data.get(KEY_NETWORK, []):
            if interface.get("name") == self._interface_name:
                return interface.get("up", False)
        return False

