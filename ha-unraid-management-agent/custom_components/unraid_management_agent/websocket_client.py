"""WebSocket client for Unraid Management Agent."""
from __future__ import annotations

import asyncio
import json
import logging
from typing import Any, Callable

import aiohttp

from .const import (
    API_WEBSOCKET,
    EVENT_ARRAY_STATUS_UPDATE,
    EVENT_CONTAINER_LIST_UPDATE,
    EVENT_DISK_LIST_UPDATE,
    EVENT_GPU_UPDATE,
    EVENT_NETWORK_LIST_UPDATE,
    EVENT_SHARE_LIST_UPDATE,
    EVENT_SYSTEM_UPDATE,
    EVENT_UPS_STATUS_UPDATE,
    EVENT_VM_LIST_UPDATE,
    WEBSOCKET_MAX_RETRIES,
    WEBSOCKET_RECONNECT_DELAY,
)

_LOGGER = logging.getLogger(__name__)


def identify_event_type(data: Any) -> str:
    """Identify event type from data structure.
    
    Events don't have a 'type' field, so we inspect the data structure.
    """
    # Handle arrays - check first element
    if isinstance(data, list):
        if not data:
            return "empty_list"
        data = data[0]
    
    # Must be a dict to identify
    if not isinstance(data, dict):
        return "unknown"
    
    # System update
    if "hostname" in data and "cpu_usage_percent" in data:
        return EVENT_SYSTEM_UPDATE
    
    # Array status
    if "state" in data and "parity_check_status" in data and "num_disks" in data:
        return EVENT_ARRAY_STATUS_UPDATE
    
    # UPS status
    if "connected" in data and "battery_charge_percent" in data:
        return EVENT_UPS_STATUS_UPDATE
    
    # GPU metrics
    if "available" in data and "driver_version" in data and "utilization_gpu_percent" in data:
        return EVENT_GPU_UPDATE
    
    # Network interface
    if "mac_address" in data and "bytes_received" in data:
        return EVENT_NETWORK_LIST_UPDATE
    
    # Container
    if "image" in data and "ports" in data and ("id" in data or "container_id" in data):
        return EVENT_CONTAINER_LIST_UPDATE
    
    # VM
    if "state" in data and "vcpus" in data:
        return EVENT_VM_LIST_UPDATE
    
    # Disk
    if "device" in data and "mount_point" in data:
        return EVENT_DISK_LIST_UPDATE
    
    # Share
    if "name" in data and "path" in data and "size_bytes" in data:
        return EVENT_SHARE_LIST_UPDATE
    
    return "unknown"


class UnraidWebSocketClient:
    """WebSocket client for real-time updates from Unraid Management Agent."""

    def __init__(
        self,
        host: str,
        port: int,
        session: aiohttp.ClientSession,
        callback: Callable[[str, Any], None],
    ) -> None:
        """Initialize the WebSocket client."""
        self.host = host
        self.port = port
        self.session = session
        self.callback = callback
        self.ws_url = f"ws://{host}:{port}{API_WEBSOCKET}"
        
        self._ws: aiohttp.ClientWebSocketResponse | None = None
        self._connected = False
        self._reconnect_count = 0
        self._stop_requested = False

    @property
    def is_connected(self) -> bool:
        """Return True if WebSocket is connected."""
        return self._connected and self._ws is not None and not self._ws.closed

    async def connect(self) -> None:
        """Connect to the WebSocket."""
        if self._stop_requested:
            return

        try:
            _LOGGER.info("Connecting to WebSocket: %s", self.ws_url)
            self._ws = await self.session.ws_connect(
                self.ws_url,
                heartbeat=30,
                timeout=aiohttp.ClientTimeout(total=10),
            )
            self._connected = True
            self._reconnect_count = 0
            _LOGGER.info("WebSocket connected successfully")
        except Exception as err:
            _LOGGER.error("Failed to connect to WebSocket: %s", err)
            self._connected = False
            raise

    async def disconnect(self) -> None:
        """Disconnect from the WebSocket."""
        self._stop_requested = True
        self._connected = False
        
        if self._ws and not self._ws.closed:
            await self._ws.close()
            _LOGGER.info("WebSocket disconnected")

    async def listen(self) -> None:
        """Listen for WebSocket messages with automatic reconnection."""
        while not self._stop_requested:
            try:
                # Connect if not connected
                if not self.is_connected:
                    await self.connect()
                
                # Listen for messages
                async for msg in self._ws:
                    if msg.type == aiohttp.WSMsgType.TEXT:
                        await self._handle_message(msg.data)
                    elif msg.type == aiohttp.WSMsgType.ERROR:
                        _LOGGER.error("WebSocket error: %s", self._ws.exception())
                        break
                    elif msg.type == aiohttp.WSMsgType.CLOSED:
                        _LOGGER.warning("WebSocket closed by server")
                        break
                
                # Connection closed, attempt reconnection
                if not self._stop_requested:
                    await self._reconnect()
                    
            except asyncio.CancelledError:
                _LOGGER.debug("WebSocket listen task cancelled")
                break
            except Exception as err:
                _LOGGER.error("WebSocket error: %s", err)
                if not self._stop_requested:
                    await self._reconnect()

    async def _handle_message(self, data: str) -> None:
        """Handle incoming WebSocket message."""
        try:
            message = json.loads(data)
            
            # Extract event data
            event_data = message.get("data")
            if event_data is None:
                _LOGGER.debug("Received message without data field")
                return
            
            # Identify event type
            event_type = identify_event_type(event_data)
            
            if event_type == "unknown":
                _LOGGER.debug("Received unknown event type, data keys: %s", 
                            list(event_data.keys()) if isinstance(event_data, dict) else type(event_data))
                return
            
            # Call callback with event type and data
            if self.callback:
                self.callback(event_type, event_data)
                
        except json.JSONDecodeError as err:
            _LOGGER.error("Failed to decode WebSocket message: %s", err)
        except Exception as err:
            _LOGGER.error("Error handling WebSocket message: %s", err)

    async def _reconnect(self) -> None:
        """Attempt to reconnect with exponential backoff."""
        self._connected = False
        
        if self._reconnect_count >= WEBSOCKET_MAX_RETRIES:
            _LOGGER.error(
                "Max reconnection attempts (%d) reached, giving up",
                WEBSOCKET_MAX_RETRIES,
            )
            self._stop_requested = True
            return
        
        # Calculate delay with exponential backoff
        delay_index = min(self._reconnect_count, len(WEBSOCKET_RECONNECT_DELAY) - 1)
        delay = WEBSOCKET_RECONNECT_DELAY[delay_index]
        
        _LOGGER.info(
            "Reconnecting in %d seconds (attempt %d/%d)",
            delay,
            self._reconnect_count + 1,
            WEBSOCKET_MAX_RETRIES,
        )
        
        await asyncio.sleep(delay)
        self._reconnect_count += 1

