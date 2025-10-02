"""Config flow for Unraid Management Agent integration."""
from __future__ import annotations

import logging
from typing import Any

import aiohttp
import voluptuous as vol

from homeassistant import config_entries
from homeassistant.const import CONF_HOST, CONF_PORT
from homeassistant.core import HomeAssistant, callback
from homeassistant.data_entry_flow import FlowResult
from homeassistant.helpers.aiohttp_client import async_get_clientsession

from .api_client import UnraidAPIClient
from .const import (
    CONF_ENABLE_WEBSOCKET,
    CONF_UPDATE_INTERVAL,
    DEFAULT_ENABLE_WEBSOCKET,
    DEFAULT_PORT,
    DEFAULT_UPDATE_INTERVAL,
    DOMAIN,
    ERROR_CANNOT_CONNECT,
    ERROR_TIMEOUT,
    ERROR_UNKNOWN,
)

_LOGGER = logging.getLogger(__name__)

STEP_USER_DATA_SCHEMA = vol.Schema(
    {
        vol.Required(CONF_HOST): str,
        vol.Required(CONF_PORT, default=DEFAULT_PORT): int,
        vol.Optional(CONF_UPDATE_INTERVAL, default=DEFAULT_UPDATE_INTERVAL): int,
        vol.Optional(CONF_ENABLE_WEBSOCKET, default=DEFAULT_ENABLE_WEBSOCKET): bool,
    }
)


async def validate_input(hass: HomeAssistant, data: dict[str, Any]) -> dict[str, Any]:
    """Validate the user input allows us to connect.

    Data has the keys from STEP_USER_DATA_SCHEMA with values provided by the user.
    """
    session = async_get_clientsession(hass)
    client = UnraidAPIClient(
        host=data[CONF_HOST],
        port=data[CONF_PORT],
        session=session,
    )

    try:
        # Test connection by getting system info
        system_info = await client.get_system_info()
        hostname = system_info.get("hostname", "unknown")
        
        return {
            "title": f"Unraid ({hostname})",
            "hostname": hostname,
        }
    except TimeoutError as err:
        _LOGGER.error("Timeout connecting to Unraid server: %s", err)
        raise TimeoutError(ERROR_TIMEOUT) from err
    except ConnectionError as err:
        _LOGGER.error("Cannot connect to Unraid server: %s", err)
        raise ConnectionError(ERROR_CANNOT_CONNECT) from err
    except Exception as err:
        _LOGGER.exception("Unexpected exception: %s", err)
        raise Exception(ERROR_UNKNOWN) from err


class UnraidConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Handle a config flow for Unraid Management Agent."""

    VERSION = 1

    async def async_step_user(
        self, user_input: dict[str, Any] | None = None
    ) -> FlowResult:
        """Handle the initial step."""
        errors: dict[str, str] = {}

        if user_input is not None:
            try:
                info = await validate_input(self.hass, user_input)
            except TimeoutError:
                errors["base"] = ERROR_TIMEOUT
            except ConnectionError:
                errors["base"] = ERROR_CANNOT_CONNECT
            except Exception:  # pylint: disable=broad-except
                _LOGGER.exception("Unexpected exception")
                errors["base"] = ERROR_UNKNOWN
            else:
                # Check if already configured
                await self.async_set_unique_id(
                    f"{user_input[CONF_HOST]}:{user_input[CONF_PORT]}"
                )
                self._abort_if_unique_id_configured()

                return self.async_create_entry(
                    title=info["title"],
                    data=user_input,
                )

        return self.async_show_form(
            step_id="user",
            data_schema=STEP_USER_DATA_SCHEMA,
            errors=errors,
        )

    @staticmethod
    @callback
    def async_get_options_flow(
        config_entry: config_entries.ConfigEntry,
    ) -> UnraidOptionsFlowHandler:
        """Get the options flow for this handler."""
        return UnraidOptionsFlowHandler(config_entry)


class UnraidOptionsFlowHandler(config_entries.OptionsFlow):
    """Handle options flow for Unraid Management Agent."""

    def __init__(self, config_entry: config_entries.ConfigEntry) -> None:
        """Initialize options flow."""
        self.config_entry = config_entry

    async def async_step_init(
        self, user_input: dict[str, Any] | None = None
    ) -> FlowResult:
        """Manage the options."""
        if user_input is not None:
            return self.async_create_entry(title="", data=user_input)

        return self.async_show_form(
            step_id="init",
            data_schema=vol.Schema(
                {
                    vol.Optional(
                        CONF_UPDATE_INTERVAL,
                        default=self.config_entry.options.get(
                            CONF_UPDATE_INTERVAL, DEFAULT_UPDATE_INTERVAL
                        ),
                    ): int,
                    vol.Optional(
                        CONF_ENABLE_WEBSOCKET,
                        default=self.config_entry.options.get(
                            CONF_ENABLE_WEBSOCKET, DEFAULT_ENABLE_WEBSOCKET
                        ),
                    ): bool,
                }
            ),
        )

