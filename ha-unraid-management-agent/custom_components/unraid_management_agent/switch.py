"""Switch platform for Unraid Management Agent."""
from __future__ import annotations

import logging
from typing import Any

from homeassistant.components.switch import SwitchEntity
from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant
from homeassistant.exceptions import HomeAssistantError
from homeassistant.helpers.entity_platform import AddEntitiesCallback
from homeassistant.helpers.update_coordinator import CoordinatorEntity

from . import UnraidDataUpdateCoordinator
from .const import (
    ATTR_CONTAINER_IMAGE,
    ATTR_CONTAINER_PORTS,
    ATTR_VM_VCPUS,
    DOMAIN,
    ERROR_CONTROL_FAILED,
    ICON_CONTAINER,
    ICON_VM,
    KEY_CONTAINERS,
    KEY_SYSTEM,
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
    """Set up Unraid switch entities."""
    coordinator: UnraidDataUpdateCoordinator = hass.data[DOMAIN][entry.entry_id]

    entities: list[SwitchEntity] = []

    # Container switches
    for container in coordinator.data.get(KEY_CONTAINERS, []):
        container_id = container.get("id") or container.get("container_id")
        container_name = container.get("name", "unknown")
        if container_id:
            entities.append(
                UnraidContainerSwitch(coordinator, entry, container_id, container_name)
            )

    # VM switches
    for vm in coordinator.data.get(KEY_VMS, []):
        vm_id = vm.get("id") or vm.get("name")
        vm_name = vm.get("name", "unknown")
        if vm_id:
            entities.append(UnraidVMSwitch(coordinator, entry, vm_id, vm_name))

    async_add_entities(entities)


class UnraidSwitchBase(CoordinatorEntity, SwitchEntity):
    """Base class for Unraid switches."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
    ) -> None:
        """Initialize the switch."""
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


# Container Switches

class UnraidContainerSwitch(UnraidSwitchBase):
    """Container control switch."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        container_id: str,
        container_name: str,
    ) -> None:
        """Initialize the switch."""
        super().__init__(coordinator, entry)
        self._container_id = container_id
        self._container_name = container_name
        self._attr_name = f"Container {container_name}"
        self._attr_icon = ICON_CONTAINER

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_container_switch_{self._container_id}"

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

    async def async_turn_on(self, **kwargs: Any) -> None:
        """Turn on the container."""
        try:
            await self.coordinator.client.start_container(self._container_id)
            _LOGGER.info("Started container: %s", self._container_name)
            # Request immediate update
            await self.coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to start container %s: %s", self._container_name, err)
            raise HomeAssistantError(
                f"{ERROR_CONTROL_FAILED}: Failed to start container {self._container_name}"
            ) from err

    async def async_turn_off(self, **kwargs: Any) -> None:
        """Turn off the container."""
        try:
            await self.coordinator.client.stop_container(self._container_id)
            _LOGGER.info("Stopped container: %s", self._container_name)
            # Request immediate update
            await self.coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to stop container %s: %s", self._container_name, err)
            raise HomeAssistantError(
                f"{ERROR_CONTROL_FAILED}: Failed to stop container {self._container_name}"
            ) from err


# VM Switches

class UnraidVMSwitch(UnraidSwitchBase):
    """VM control switch."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
        vm_id: str,
        vm_name: str,
    ) -> None:
        """Initialize the switch."""
        super().__init__(coordinator, entry)
        self._vm_id = vm_id
        self._vm_name = vm_name
        self._attr_name = f"VM {vm_name}"
        self._attr_icon = ICON_VM

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_vm_switch_{self._vm_id}"

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

    async def async_turn_on(self, **kwargs: Any) -> None:
        """Turn on the VM."""
        try:
            await self.coordinator.client.start_vm(self._vm_id)
            _LOGGER.info("Started VM: %s", self._vm_name)
            # Request immediate update
            await self.coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to start VM %s: %s", self._vm_name, err)
            raise HomeAssistantError(
                f"{ERROR_CONTROL_FAILED}: Failed to start VM {self._vm_name}"
            ) from err

    async def async_turn_off(self, **kwargs: Any) -> None:
        """Turn off the VM."""
        try:
            await self.coordinator.client.stop_vm(self._vm_id)
            _LOGGER.info("Stopped VM: %s", self._vm_name)
            # Request immediate update
            await self.coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to stop VM %s: %s", self._vm_name, err)
            raise HomeAssistantError(
                f"{ERROR_CONTROL_FAILED}: Failed to stop VM {self._vm_name}"
            ) from err

