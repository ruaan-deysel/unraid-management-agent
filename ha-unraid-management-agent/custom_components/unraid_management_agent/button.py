"""Button platform for Unraid Management Agent."""
from __future__ import annotations

import logging
from typing import Any

from homeassistant.components.button import ButtonEntity
from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant
from homeassistant.exceptions import HomeAssistantError
from homeassistant.helpers.entity_platform import AddEntitiesCallback
from homeassistant.helpers.update_coordinator import CoordinatorEntity

from . import UnraidDataUpdateCoordinator
from .const import (
    DOMAIN,
    ERROR_CONTROL_FAILED,
    ICON_ARRAY,
    ICON_PARITY,
    KEY_SYSTEM,
    MANUFACTURER,
    MODEL,
)

_LOGGER = logging.getLogger(__name__)


async def async_setup_entry(
    hass: HomeAssistant,
    entry: ConfigEntry,
    async_add_entities: AddEntitiesCallback,
) -> None:
    """Set up Unraid button entities."""
    coordinator: UnraidDataUpdateCoordinator = hass.data[DOMAIN][entry.entry_id]

    entities: list[ButtonEntity] = [
        UnraidArrayStartButton(coordinator, entry),
        UnraidArrayStopButton(coordinator, entry),
        UnraidParityCheckStartButton(coordinator, entry),
        UnraidParityCheckStopButton(coordinator, entry),
    ]

    async_add_entities(entities)


class UnraidButtonBase(CoordinatorEntity, ButtonEntity):
    """Base class for Unraid buttons."""

    def __init__(
        self,
        coordinator: UnraidDataUpdateCoordinator,
        entry: ConfigEntry,
    ) -> None:
        """Initialize the button."""
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


# Array Control Buttons

class UnraidArrayStartButton(UnraidButtonBase):
    """Array start button."""

    _attr_name = "Start Array"
    _attr_icon = ICON_ARRAY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_array_start_button"

    async def async_press(self) -> None:
        """Handle the button press."""
        try:
            await self.coordinator.client.start_array()
            _LOGGER.info("Array start command sent")
            # Request immediate update
            await self.coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to start array: %s", err)
            raise HomeAssistantError(
                f"{ERROR_CONTROL_FAILED}: Failed to start array"
            ) from err


class UnraidArrayStopButton(UnraidButtonBase):
    """Array stop button."""

    _attr_name = "Stop Array"
    _attr_icon = ICON_ARRAY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_array_stop_button"

    async def async_press(self) -> None:
        """Handle the button press."""
        try:
            await self.coordinator.client.stop_array()
            _LOGGER.info("Array stop command sent")
            # Request immediate update
            await self.coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to stop array: %s", err)
            raise HomeAssistantError(
                f"{ERROR_CONTROL_FAILED}: Failed to stop array"
            ) from err


# Parity Check Control Buttons

class UnraidParityCheckStartButton(UnraidButtonBase):
    """Parity check start button."""

    _attr_name = "Start Parity Check"
    _attr_icon = ICON_PARITY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_parity_check_start_button"

    async def async_press(self) -> None:
        """Handle the button press."""
        try:
            await self.coordinator.client.start_parity_check()
            _LOGGER.info("Parity check start command sent")
            # Request immediate update
            await self.coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to start parity check: %s", err)
            raise HomeAssistantError(
                f"{ERROR_CONTROL_FAILED}: Failed to start parity check"
            ) from err


class UnraidParityCheckStopButton(UnraidButtonBase):
    """Parity check stop button."""

    _attr_name = "Stop Parity Check"
    _attr_icon = ICON_PARITY

    @property
    def unique_id(self) -> str:
        """Return unique ID."""
        return f"{self._entry.entry_id}_parity_check_stop_button"

    async def async_press(self) -> None:
        """Handle the button press."""
        try:
            await self.coordinator.client.stop_parity_check()
            _LOGGER.info("Parity check stop command sent")
            # Request immediate update
            await self.coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to stop parity check: %s", err)
            raise HomeAssistantError(
                f"{ERROR_CONTROL_FAILED}: Failed to stop parity check"
            ) from err

