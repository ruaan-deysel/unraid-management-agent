"""Repair flows for Unraid Management Agent integration."""
from __future__ import annotations

import logging
from typing import Any

from homeassistant import data_entry_flow
from homeassistant.components.repairs import RepairsFlow
from homeassistant.core import HomeAssistant
from homeassistant.helpers import issue_registry as ir

from .const import DOMAIN

_LOGGER = logging.getLogger(__name__)


async def async_create_fix_flow(
    hass: HomeAssistant,
    issue_id: str,
    data: dict[str, str | int | float | None] | None,
) -> RepairsFlow:
    """Create a fix flow for an issue."""
    if issue_id.startswith("connection_"):
        return ConnectionIssueRepairFlow(hass, issue_id, data)
    if issue_id.startswith("disk_health_"):
        return DiskHealthRepairFlow(hass, issue_id, data)
    if issue_id.startswith("array_"):
        return ArrayIssueRepairFlow(hass, issue_id, data)
    if issue_id.startswith("parity_"):
        return ParityCheckRepairFlow(hass, issue_id, data)
    return RepairsFlow()


class ConnectionIssueRepairFlow(RepairsFlow):
    """Handler for connection issue repairs."""

    def __init__(
        self,
        hass: HomeAssistant,
        issue_id: str,
        data: dict[str, str | int | float | None] | None,
    ) -> None:
        """Initialize the repair flow."""
        super().__init__()
        self.hass = hass
        self.issue_id = issue_id
        self.data = data or {}

    async def async_step_init(
        self, user_input: dict[str, Any] | None = None
    ) -> data_entry_flow.FlowResult:
        """Handle the initial step."""
        if user_input is not None:
            # Mark issue as resolved
            ir.async_delete_issue(self.hass, DOMAIN, self.issue_id)
            return self.async_create_entry(title="", data={})

        return self.async_show_form(
            step_id="init",
            description_placeholders={
                "error": self.data.get("error", "Unknown error"),
                "host": self.data.get("host", "Unknown"),
                "port": str(self.data.get("port", "Unknown")),
            },
        )


class DiskHealthRepairFlow(RepairsFlow):
    """Handler for disk health issue repairs."""

    def __init__(
        self,
        hass: HomeAssistant,
        issue_id: str,
        data: dict[str, str | int | float | None] | None,
    ) -> None:
        """Initialize the repair flow."""
        super().__init__()
        self.hass = hass
        self.issue_id = issue_id
        self.data = data or {}

    async def async_step_init(
        self, user_input: dict[str, Any] | None = None
    ) -> data_entry_flow.FlowResult:
        """Handle the initial step."""
        if user_input is not None:
            # Mark issue as resolved
            ir.async_delete_issue(self.hass, DOMAIN, self.issue_id)
            return self.async_create_entry(title="", data={})

        return self.async_show_form(
            step_id="init",
            description_placeholders={
                "disk_name": self.data.get("disk_name", "Unknown"),
                "smart_status": self.data.get("smart_status", "Unknown"),
                "smart_errors": str(self.data.get("smart_errors", 0)),
                "temperature": str(self.data.get("temperature", "Unknown")),
            },
        )


class ArrayIssueRepairFlow(RepairsFlow):
    """Handler for array issue repairs."""

    def __init__(
        self,
        hass: HomeAssistant,
        issue_id: str,
        data: dict[str, str | int | float | None] | None,
    ) -> None:
        """Initialize the repair flow."""
        super().__init__()
        self.hass = hass
        self.issue_id = issue_id
        self.data = data or {}

    async def async_step_init(
        self, user_input: dict[str, Any] | None = None
    ) -> data_entry_flow.FlowResult:
        """Handle the initial step."""
        if user_input is not None:
            # Mark issue as resolved
            ir.async_delete_issue(self.hass, DOMAIN, self.issue_id)
            return self.async_create_entry(title="", data={})

        return self.async_show_form(
            step_id="init",
            description_placeholders={
                "array_state": self.data.get("array_state", "Unknown"),
                "issue_description": self.data.get("issue_description", "Unknown issue"),
            },
        )


class ParityCheckRepairFlow(RepairsFlow):
    """Handler for parity check issue repairs."""

    def __init__(
        self,
        hass: HomeAssistant,
        issue_id: str,
        data: dict[str, str | int | float | None] | None,
    ) -> None:
        """Initialize the repair flow."""
        super().__init__()
        self.hass = hass
        self.issue_id = issue_id
        self.data = data or {}

    async def async_step_init(
        self, user_input: dict[str, Any] | None = None
    ) -> data_entry_flow.FlowResult:
        """Handle the initial step."""
        if user_input is not None:
            # Mark issue as resolved
            ir.async_delete_issue(self.hass, DOMAIN, self.issue_id)
            return self.async_create_entry(title="", data={})

        return self.async_show_form(
            step_id="init",
            description_placeholders={
                "parity_status": self.data.get("parity_status", "Unknown"),
                "sync_percent": str(self.data.get("sync_percent", 0)),
                "errors_found": str(self.data.get("errors_found", 0)),
            },
        )


async def async_check_and_create_issues(
    hass: HomeAssistant, coordinator
) -> None:
    """Check for issues and create repair flows if needed."""
    
    # Check for connection issues
    if not coordinator.last_update_success:
        ir.async_create_issue(
            hass,
            DOMAIN,
            f"connection_{coordinator.config_entry.entry_id}",
            is_fixable=True,
            severity=ir.IssueSeverity.ERROR,
            translation_key="connection_failed",
            translation_placeholders={
                "host": coordinator.config_entry.data.get("host", "Unknown"),
                "port": str(coordinator.config_entry.data.get("port", "Unknown")),
                "error": str(coordinator.last_exception) if coordinator.last_exception else "Unknown error",
            },
        )
    
    # Check for disk health issues
    disks = coordinator.data.get("disks", [])
    for disk in disks:
        disk_id = disk.get("id", disk.get("name", "unknown"))
        smart_errors = disk.get("smart_errors", 0)
        smart_status = disk.get("smart_status", "UNKNOWN")
        temperature = disk.get("temperature_celsius", 0)
        
        # Check for SMART errors
        if smart_errors > 0:
            ir.async_create_issue(
                hass,
                DOMAIN,
                f"disk_health_{disk_id}_smart_errors",
                is_fixable=True,
                severity=ir.IssueSeverity.WARNING,
                translation_key="disk_smart_errors",
                translation_placeholders={
                    "disk_name": disk.get("name", disk_id),
                    "smart_errors": str(smart_errors),
                    "smart_status": smart_status,
                },
            )
        
        # Check for high temperature (>50°C)
        if temperature > 50:
            ir.async_create_issue(
                hass,
                DOMAIN,
                f"disk_health_{disk_id}_high_temp",
                is_fixable=True,
                severity=ir.IssueSeverity.WARNING,
                translation_key="disk_high_temperature",
                translation_placeholders={
                    "disk_name": disk.get("name", disk_id),
                    "temperature": str(temperature),
                },
            )
    
    # Check for array issues
    array_data = coordinator.data.get("array", {})
    parity_valid = array_data.get("parity_valid", True)
    
    if not parity_valid:
        ir.async_create_issue(
            hass,
            DOMAIN,
            f"array_parity_invalid_{coordinator.config_entry.entry_id}",
            is_fixable=True,
            severity=ir.IssueSeverity.ERROR,
            translation_key="array_parity_invalid",
            translation_placeholders={
                "array_state": array_data.get("state", "Unknown"),
            },
        )
    
    # Check for parity check issues
    parity_check_running = array_data.get("parity_check_running", False)
    sync_percent = array_data.get("sync_percent", 0)
    
    # If parity check has been running for a very long time (>95% but not complete)
    if parity_check_running and sync_percent > 95 and sync_percent < 100:
        ir.async_create_issue(
            hass,
            DOMAIN,
            f"parity_check_stuck_{coordinator.config_entry.entry_id}",
            is_fixable=True,
            severity=ir.IssueSeverity.WARNING,
            translation_key="parity_check_stuck",
            translation_placeholders={
                "sync_percent": str(sync_percent),
            },
        )

