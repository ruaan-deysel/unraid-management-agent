"""The Unraid Management Agent integration."""
from __future__ import annotations

import asyncio
import logging
from datetime import timedelta
from typing import Any

from homeassistant.config_entries import ConfigEntry
from homeassistant.const import CONF_HOST, CONF_PORT, Platform
from homeassistant.core import HomeAssistant
from homeassistant.exceptions import ConfigEntryNotReady
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.update_coordinator import (
    DataUpdateCoordinator,
    UpdateFailed,
)

from .api_client import UnraidAPIClient
from .const import (
    CONF_ENABLE_WEBSOCKET,
    CONF_UPDATE_INTERVAL,
    DEFAULT_ENABLE_WEBSOCKET,
    DEFAULT_UPDATE_INTERVAL,
    DOMAIN,
    KEY_ARRAY,
    KEY_CONTAINERS,
    KEY_GPU,
    KEY_NETWORK,
    KEY_SYSTEM,
    KEY_UPS,
    KEY_VMS,
)

_LOGGER = logging.getLogger(__name__)

PLATFORMS: list[Platform] = [
    Platform.SENSOR,
    Platform.BINARY_SENSOR,
    Platform.SWITCH,
    Platform.BUTTON,
]


async def async_setup_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Set up Unraid Management Agent from a config entry."""
    host = entry.data[CONF_HOST]
    port = entry.data[CONF_PORT]
    update_interval = entry.options.get(CONF_UPDATE_INTERVAL, DEFAULT_UPDATE_INTERVAL)
    enable_websocket = entry.options.get(CONF_ENABLE_WEBSOCKET, DEFAULT_ENABLE_WEBSOCKET)

    session = async_get_clientsession(hass)
    client = UnraidAPIClient(host=host, port=port, session=session)

    # Test connection
    try:
        await client.health_check()
    except Exception as err:
        _LOGGER.error("Failed to connect to Unraid server: %s", err)
        raise ConfigEntryNotReady from err

    # Create coordinator
    coordinator = UnraidDataUpdateCoordinator(
        hass,
        client=client,
        update_interval=update_interval,
        enable_websocket=enable_websocket,
    )

    # Fetch initial data
    await coordinator.async_config_entry_first_refresh()

    # Store coordinator
    hass.data.setdefault(DOMAIN, {})
    hass.data[DOMAIN][entry.entry_id] = coordinator

    # Set up platforms
    await hass.config_entries.async_forward_entry_setups(entry, PLATFORMS)

    # Register update listener for options
    entry.async_on_unload(entry.add_update_listener(async_reload_entry))

    return True


async def async_unload_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Unload a config entry."""
    if unload_ok := await hass.config_entries.async_unload_platforms(entry, PLATFORMS):
        coordinator: UnraidDataUpdateCoordinator = hass.data[DOMAIN].pop(entry.entry_id)
        # Stop WebSocket if running
        if hasattr(coordinator, "websocket_task") and coordinator.websocket_task:
            coordinator.websocket_task.cancel()
            try:
                await coordinator.websocket_task
            except asyncio.CancelledError:
                pass

    return unload_ok


async def async_reload_entry(hass: HomeAssistant, entry: ConfigEntry) -> None:
    """Reload config entry when options change."""
    await hass.config_entries.async_reload(entry.entry_id)


class UnraidDataUpdateCoordinator(DataUpdateCoordinator):
    """Class to manage fetching Unraid data from the API."""

    def __init__(
        self,
        hass: HomeAssistant,
        client: UnraidAPIClient,
        update_interval: int,
        enable_websocket: bool,
    ) -> None:
        """Initialize the coordinator."""
        self.client = client
        self.enable_websocket = enable_websocket
        self.websocket_task = None

        super().__init__(
            hass,
            _LOGGER,
            name=DOMAIN,
            update_interval=timedelta(seconds=update_interval),
        )

    async def _async_update_data(self) -> dict[str, Any]:
        """Fetch data from API endpoint.

        This is the place to pre-process the data to lookup tables
        so entities can quickly look up their data.
        """
        try:
            # Fetch all data in parallel
            results = await asyncio.gather(
                self.client.get_system_info(),
                self.client.get_array_status(),
                self.client.get_containers(),
                self.client.get_vms(),
                self.client.get_ups_status(),
                self.client.get_gpu_metrics(),
                self.client.get_network_interfaces(),
                return_exceptions=True,
            )

            # Process results
            data = {
                KEY_SYSTEM: results[0] if not isinstance(results[0], Exception) else {},
                KEY_ARRAY: results[1] if not isinstance(results[1], Exception) else {},
                KEY_CONTAINERS: results[2] if not isinstance(results[2], Exception) else [],
                KEY_VMS: results[3] if not isinstance(results[3], Exception) else [],
                KEY_UPS: results[4] if not isinstance(results[4], Exception) else {},
                KEY_GPU: results[5] if not isinstance(results[5], Exception) else [],
                KEY_NETWORK: results[6] if not isinstance(results[6], Exception) else [],
            }

            # Log any errors
            for i, result in enumerate(results):
                if isinstance(result, Exception):
                    _LOGGER.warning("Error fetching data for index %d: %s", i, result)

            return data

        except Exception as err:
            _LOGGER.error("Error communicating with API: %s", err)
            raise UpdateFailed(f"Error communicating with API: {err}") from err

    async def async_start_websocket(self) -> None:
        """Start WebSocket connection for real-time updates."""
        if not self.enable_websocket:
            _LOGGER.debug("WebSocket disabled in configuration")
            return

        # WebSocket implementation will be added in Phase 2.2
        _LOGGER.info("WebSocket support will be added in Phase 2.2")

    async def async_stop_websocket(self) -> None:
        """Stop WebSocket connection."""
        if self.websocket_task:
            self.websocket_task.cancel()
            try:
                await self.websocket_task
            except asyncio.CancelledError:
                pass
            self.websocket_task = None

