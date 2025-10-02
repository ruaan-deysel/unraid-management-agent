"""The Unraid Management Agent integration."""
from __future__ import annotations

import asyncio
import logging
from datetime import timedelta
from typing import Any

from homeassistant.config_entries import ConfigEntry
from homeassistant.const import CONF_HOST, CONF_PORT, Platform
from homeassistant.core import HomeAssistant, ServiceCall
from homeassistant.exceptions import ConfigEntryNotReady, HomeAssistantError
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.update_coordinator import (
    DataUpdateCoordinator,
    UpdateFailed,
)

from .api_client import UnraidAPIClient
from . import repairs
from .const import (
    CONF_ENABLE_WEBSOCKET,
    CONF_UPDATE_INTERVAL,
    DEFAULT_ENABLE_WEBSOCKET,
    DEFAULT_UPDATE_INTERVAL,
    DOMAIN,
    EVENT_ARRAY_STATUS_UPDATE,
    EVENT_CONTAINER_LIST_UPDATE,
    EVENT_DISK_LIST_UPDATE,
    EVENT_GPU_UPDATE,
    EVENT_NETWORK_LIST_UPDATE,
    EVENT_SYSTEM_UPDATE,
    EVENT_UPS_STATUS_UPDATE,
    EVENT_VM_LIST_UPDATE,
    KEY_ARRAY,
    KEY_CONTAINERS,
    KEY_DISKS,
    KEY_GPU,
    KEY_NETWORK,
    KEY_SYSTEM,
    KEY_UPS,
    KEY_VMS,
)
from .websocket_client import UnraidWebSocketClient

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

    # Register services
    await async_setup_services(hass, coordinator)

    # Start WebSocket for real-time updates
    if enable_websocket:
        await coordinator.async_start_websocket()

    # Register update listener for options
    entry.async_on_unload(entry.add_update_listener(async_reload_entry))

    return True


async def async_unload_entry(hass: HomeAssistant, entry: ConfigEntry) -> bool:
    """Unload a config entry."""
    if unload_ok := await hass.config_entries.async_unload_platforms(entry, PLATFORMS):
        coordinator: UnraidDataUpdateCoordinator = hass.data[DOMAIN].pop(entry.entry_id)
        # Stop WebSocket if running
        await coordinator.async_stop_websocket()

    return unload_ok


async def async_reload_entry(hass: HomeAssistant, entry: ConfigEntry) -> None:
    """Reload config entry when options change."""
    await hass.config_entries.async_reload(entry.entry_id)


async def async_setup_services(
    hass: HomeAssistant, coordinator: UnraidDataUpdateCoordinator
) -> None:
    """Set up services for Unraid Management Agent."""

    async def handle_container_start(call: ServiceCall) -> None:
        """Handle container start service."""
        container_id = call.data["container_id"]
        try:
            await coordinator.client.start_container(container_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to start container %s: %s", container_id, err)
            raise HomeAssistantError(f"Failed to start container: {err}") from err

    async def handle_container_stop(call: ServiceCall) -> None:
        """Handle container stop service."""
        container_id = call.data["container_id"]
        try:
            await coordinator.client.stop_container(container_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to stop container %s: %s", container_id, err)
            raise HomeAssistantError(f"Failed to stop container: {err}") from err

    async def handle_container_restart(call: ServiceCall) -> None:
        """Handle container restart service."""
        container_id = call.data["container_id"]
        try:
            await coordinator.client.restart_container(container_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to restart container %s: %s", container_id, err)
            raise HomeAssistantError(f"Failed to restart container: {err}") from err

    async def handle_container_pause(call: ServiceCall) -> None:
        """Handle container pause service."""
        container_id = call.data["container_id"]
        try:
            await coordinator.client.pause_container(container_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to pause container %s: %s", container_id, err)
            raise HomeAssistantError(f"Failed to pause container: {err}") from err

    async def handle_container_resume(call: ServiceCall) -> None:
        """Handle container resume service."""
        container_id = call.data["container_id"]
        try:
            await coordinator.client.unpause_container(container_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to resume container %s: %s", container_id, err)
            raise HomeAssistantError(f"Failed to resume container: {err}") from err

    async def handle_vm_start(call: ServiceCall) -> None:
        """Handle VM start service."""
        vm_id = call.data["vm_id"]
        try:
            await coordinator.client.start_vm(vm_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to start VM %s: %s", vm_id, err)
            raise HomeAssistantError(f"Failed to start VM: {err}") from err

    async def handle_vm_stop(call: ServiceCall) -> None:
        """Handle VM stop service."""
        vm_id = call.data["vm_id"]
        try:
            await coordinator.client.stop_vm(vm_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to stop VM %s: %s", vm_id, err)
            raise HomeAssistantError(f"Failed to stop VM: {err}") from err

    async def handle_vm_restart(call: ServiceCall) -> None:
        """Handle VM restart service."""
        vm_id = call.data["vm_id"]
        try:
            await coordinator.client.restart_vm(vm_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to restart VM %s: %s", vm_id, err)
            raise HomeAssistantError(f"Failed to restart VM: {err}") from err

    async def handle_vm_pause(call: ServiceCall) -> None:
        """Handle VM pause service."""
        vm_id = call.data["vm_id"]
        try:
            await coordinator.client.pause_vm(vm_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to pause VM %s: %s", vm_id, err)
            raise HomeAssistantError(f"Failed to pause VM: {err}") from err

    async def handle_vm_resume(call: ServiceCall) -> None:
        """Handle VM resume service."""
        vm_id = call.data["vm_id"]
        try:
            await coordinator.client.resume_vm(vm_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to resume VM %s: %s", vm_id, err)
            raise HomeAssistantError(f"Failed to resume VM: {err}") from err

    async def handle_vm_hibernate(call: ServiceCall) -> None:
        """Handle VM hibernate service."""
        vm_id = call.data["vm_id"]
        try:
            await coordinator.client.hibernate_vm(vm_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to hibernate VM %s: %s", vm_id, err)
            raise HomeAssistantError(f"Failed to hibernate VM: {err}") from err

    async def handle_vm_force_stop(call: ServiceCall) -> None:
        """Handle VM force stop service."""
        vm_id = call.data["vm_id"]
        try:
            await coordinator.client.force_stop_vm(vm_id)
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to force stop VM %s: %s", vm_id, err)
            raise HomeAssistantError(f"Failed to force stop VM: {err}") from err

    async def handle_array_start(call: ServiceCall) -> None:
        """Handle array start service."""
        try:
            await coordinator.client.start_array()
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to start array: %s", err)
            raise HomeAssistantError(f"Failed to start array: {err}") from err

    async def handle_array_stop(call: ServiceCall) -> None:
        """Handle array stop service."""
        try:
            await coordinator.client.stop_array()
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to stop array: %s", err)
            raise HomeAssistantError(f"Failed to stop array: {err}") from err

    async def handle_parity_check_start(call: ServiceCall) -> None:
        """Handle parity check start service."""
        try:
            await coordinator.client.start_parity_check()
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to start parity check: %s", err)
            raise HomeAssistantError(f"Failed to start parity check: {err}") from err

    async def handle_parity_check_stop(call: ServiceCall) -> None:
        """Handle parity check stop service."""
        try:
            await coordinator.client.stop_parity_check()
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to stop parity check: %s", err)
            raise HomeAssistantError(f"Failed to stop parity check: {err}") from err

    async def handle_parity_check_pause(call: ServiceCall) -> None:
        """Handle parity check pause service."""
        try:
            await coordinator.client.pause_parity_check()
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to pause parity check: %s", err)
            raise HomeAssistantError(f"Failed to pause parity check: {err}") from err

    async def handle_parity_check_resume(call: ServiceCall) -> None:
        """Handle parity check resume service."""
        try:
            await coordinator.client.resume_parity_check()
            await coordinator.async_request_refresh()
        except Exception as err:
            _LOGGER.error("Failed to resume parity check: %s", err)
            raise HomeAssistantError(f"Failed to resume parity check: {err}") from err

    # Register all services
    hass.services.async_register(DOMAIN, "container_start", handle_container_start)
    hass.services.async_register(DOMAIN, "container_stop", handle_container_stop)
    hass.services.async_register(DOMAIN, "container_restart", handle_container_restart)
    hass.services.async_register(DOMAIN, "container_pause", handle_container_pause)
    hass.services.async_register(DOMAIN, "container_resume", handle_container_resume)

    hass.services.async_register(DOMAIN, "vm_start", handle_vm_start)
    hass.services.async_register(DOMAIN, "vm_stop", handle_vm_stop)
    hass.services.async_register(DOMAIN, "vm_restart", handle_vm_restart)
    hass.services.async_register(DOMAIN, "vm_pause", handle_vm_pause)
    hass.services.async_register(DOMAIN, "vm_resume", handle_vm_resume)
    hass.services.async_register(DOMAIN, "vm_hibernate", handle_vm_hibernate)
    hass.services.async_register(DOMAIN, "vm_force_stop", handle_vm_force_stop)

    hass.services.async_register(DOMAIN, "array_start", handle_array_start)
    hass.services.async_register(DOMAIN, "array_stop", handle_array_stop)

    hass.services.async_register(DOMAIN, "parity_check_start", handle_parity_check_start)
    hass.services.async_register(DOMAIN, "parity_check_stop", handle_parity_check_stop)
    hass.services.async_register(DOMAIN, "parity_check_pause", handle_parity_check_pause)
    hass.services.async_register(DOMAIN, "parity_check_resume", handle_parity_check_resume)

    _LOGGER.info("Registered %d services for Unraid Management Agent", 18)


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
                self.client.get_disks(),
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
                KEY_DISKS: results[2] if not isinstance(results[2], Exception) else [],
                KEY_CONTAINERS: results[3] if not isinstance(results[3], Exception) else [],
                KEY_VMS: results[4] if not isinstance(results[4], Exception) else [],
                KEY_UPS: results[5] if not isinstance(results[5], Exception) else {},
                KEY_GPU: results[6] if not isinstance(results[6], Exception) else [],
                KEY_NETWORK: results[7] if not isinstance(results[7], Exception) else [],
            }

            # Log any errors
            for i, result in enumerate(results):
                if isinstance(result, Exception):
                    _LOGGER.warning("Error fetching data for index %d: %s", i, result)

            # Check for issues and create repair flows
            await repairs.async_check_and_create_issues(self.hass, self)

            return data

        except Exception as err:
            _LOGGER.error("Error communicating with API: %s", err)
            raise UpdateFailed(f"Error communicating with API: {err}") from err

    def _handle_websocket_event(self, event_type: str, data: Any) -> None:
        """Handle WebSocket event and update coordinator data."""
        if not self.data:
            return

        # Update coordinator data based on event type
        if event_type == EVENT_SYSTEM_UPDATE:
            self.data[KEY_SYSTEM] = data
        elif event_type == EVENT_ARRAY_STATUS_UPDATE:
            self.data[KEY_ARRAY] = data
        elif event_type == EVENT_DISK_LIST_UPDATE:
            self.data[KEY_DISKS] = data if isinstance(data, list) else [data]
        elif event_type == EVENT_UPS_STATUS_UPDATE:
            self.data[KEY_UPS] = data
        elif event_type == EVENT_GPU_UPDATE:
            self.data[KEY_GPU] = data if isinstance(data, list) else [data]
        elif event_type == EVENT_NETWORK_LIST_UPDATE:
            self.data[KEY_NETWORK] = data if isinstance(data, list) else [data]
        elif event_type == EVENT_CONTAINER_LIST_UPDATE:
            self.data[KEY_CONTAINERS] = data if isinstance(data, list) else [data]
        elif event_type == EVENT_VM_LIST_UPDATE:
            self.data[KEY_VMS] = data if isinstance(data, list) else [data]

        # Notify listeners of data update
        self.async_set_updated_data(self.data)

    async def async_start_websocket(self) -> None:
        """Start WebSocket connection for real-time updates."""
        if not self.enable_websocket:
            _LOGGER.debug("WebSocket disabled in configuration")
            return

        if self.websocket_task and not self.websocket_task.done():
            _LOGGER.debug("WebSocket already running")
            return

        try:
            # Create WebSocket client
            ws_client = UnraidWebSocketClient(
                host=self.client.host,
                port=self.client.port,
                session=self.client.session,
                callback=self._handle_websocket_event,
            )

            # Start listening in background task
            self.websocket_task = asyncio.create_task(ws_client.listen())
            _LOGGER.info("WebSocket client started")

        except Exception as err:
            _LOGGER.error("Failed to start WebSocket client: %s", err)
            self.websocket_task = None

    async def async_stop_websocket(self) -> None:
        """Stop WebSocket connection."""
        if self.websocket_task:
            self.websocket_task.cancel()
            try:
                await self.websocket_task
            except asyncio.CancelledError:
                pass
            self.websocket_task = None
            _LOGGER.info("WebSocket client stopped")

