"""Constants for the Unraid Management Agent integration."""
from datetime import timedelta
from typing import Final

# Integration domain
DOMAIN: Final = "unraid_management_agent"

# Configuration keys
CONF_HOST: Final = "host"
CONF_PORT: Final = "port"
CONF_UPDATE_INTERVAL: Final = "update_interval"
CONF_ENABLE_WEBSOCKET: Final = "enable_websocket"

# Default values
DEFAULT_PORT: Final = 8043
DEFAULT_UPDATE_INTERVAL: Final = 30  # seconds
DEFAULT_ENABLE_WEBSOCKET: Final = True

# Update intervals
UPDATE_INTERVAL: Final = timedelta(seconds=DEFAULT_UPDATE_INTERVAL)
WEBSOCKET_RECONNECT_DELAY: Final = [1, 2, 4, 8, 16, 32, 60]  # Exponential backoff in seconds
WEBSOCKET_MAX_RETRIES: Final = 10

# API endpoints
API_BASE: Final = "/api/v1"
API_HEALTH: Final = f"{API_BASE}/health"
API_SYSTEM: Final = f"{API_BASE}/system"
API_ARRAY: Final = f"{API_BASE}/array"
API_DISKS: Final = f"{API_BASE}/disks"
API_SHARES: Final = f"{API_BASE}/shares"
API_DOCKER: Final = f"{API_BASE}/docker"
API_VM: Final = f"{API_BASE}/vm"
API_UPS: Final = f"{API_BASE}/ups"
API_GPU: Final = f"{API_BASE}/gpu"
API_NETWORK: Final = f"{API_BASE}/network"
API_WEBSOCKET: Final = f"{API_BASE}/ws"

# Control endpoints
API_DOCKER_START: Final = f"{API_DOCKER}/{{id}}/start"
API_DOCKER_STOP: Final = f"{API_DOCKER}/{{id}}/stop"
API_DOCKER_RESTART: Final = f"{API_DOCKER}/{{id}}/restart"
API_DOCKER_PAUSE: Final = f"{API_DOCKER}/{{id}}/pause"
API_DOCKER_UNPAUSE: Final = f"{API_DOCKER}/{{id}}/unpause"

API_VM_START: Final = f"{API_VM}/{{id}}/start"
API_VM_STOP: Final = f"{API_VM}/{{id}}/stop"
API_VM_RESTART: Final = f"{API_VM}/{{id}}/restart"
API_VM_PAUSE: Final = f"{API_VM}/{{id}}/pause"
API_VM_RESUME: Final = f"{API_VM}/{{id}}/resume"
API_VM_HIBERNATE: Final = f"{API_VM}/{{id}}/hibernate"
API_VM_FORCE_STOP: Final = f"{API_VM}/{{id}}/force-stop"

API_ARRAY_START: Final = f"{API_ARRAY}/start"
API_ARRAY_STOP: Final = f"{API_ARRAY}/stop"
API_PARITY_CHECK_START: Final = f"{API_ARRAY}/parity-check/start"
API_PARITY_CHECK_STOP: Final = f"{API_ARRAY}/parity-check/stop"
API_PARITY_CHECK_PAUSE: Final = f"{API_ARRAY}/parity-check/pause"
API_PARITY_CHECK_RESUME: Final = f"{API_ARRAY}/parity-check/resume"

# Event types
EVENT_SYSTEM_UPDATE: Final = "system_update"
EVENT_ARRAY_STATUS_UPDATE: Final = "array_status_update"
EVENT_DISK_LIST_UPDATE: Final = "disk_list_update"
EVENT_SHARE_LIST_UPDATE: Final = "share_list_update"
EVENT_CONTAINER_LIST_UPDATE: Final = "container_list_update"
EVENT_VM_LIST_UPDATE: Final = "vm_list_update"
EVENT_UPS_STATUS_UPDATE: Final = "ups_status_update"
EVENT_GPU_UPDATE: Final = "gpu_update"
EVENT_NETWORK_LIST_UPDATE: Final = "network_list_update"

# Entity keys
KEY_SYSTEM: Final = "system"
KEY_ARRAY: Final = "array"
KEY_DISKS: Final = "disks"
KEY_SHARES: Final = "shares"
KEY_CONTAINERS: Final = "containers"
KEY_VMS: Final = "vms"
KEY_UPS: Final = "ups"
KEY_GPU: Final = "gpu"
KEY_NETWORK: Final = "network"

# Sensor types
SENSOR_CPU_USAGE: Final = "cpu_usage"
SENSOR_RAM_USAGE: Final = "ram_usage"
SENSOR_CPU_TEMP: Final = "cpu_temperature"
SENSOR_UPTIME: Final = "uptime"
SENSOR_ARRAY_USAGE: Final = "array_usage"
SENSOR_PARITY_PROGRESS: Final = "parity_progress"
SENSOR_GPU_NAME: Final = "gpu_name"
SENSOR_GPU_UTILIZATION: Final = "gpu_utilization"
SENSOR_GPU_CPU_TEMP: Final = "gpu_cpu_temperature"
SENSOR_GPU_POWER: Final = "gpu_power"
SENSOR_UPS_BATTERY: Final = "ups_battery"
SENSOR_UPS_LOAD: Final = "ups_load"
SENSOR_UPS_RUNTIME: Final = "ups_runtime"

# Binary sensor types
BINARY_SENSOR_ARRAY_STARTED: Final = "array_started"
BINARY_SENSOR_PARITY_CHECK_RUNNING: Final = "parity_check_running"
BINARY_SENSOR_PARITY_VALID: Final = "parity_valid"
BINARY_SENSOR_UPS_CONNECTED: Final = "ups_connected"
BINARY_SENSOR_CONTAINER_RUNNING: Final = "container_running"
BINARY_SENSOR_VM_RUNNING: Final = "vm_running"
BINARY_SENSOR_NETWORK_UP: Final = "network_up"

# Switch types
SWITCH_CONTAINER: Final = "container"
SWITCH_VM: Final = "vm"

# Button types
BUTTON_ARRAY_START: Final = "array_start"
BUTTON_ARRAY_STOP: Final = "array_stop"
BUTTON_PARITY_CHECK_START: Final = "parity_check_start"
BUTTON_PARITY_CHECK_STOP: Final = "parity_check_stop"
BUTTON_CONTAINER_RESTART: Final = "container_restart"

# Device info
MANUFACTURER: Final = "Lime Technology"
MODEL: Final = "Unraid Server"

# Attributes
ATTR_HOSTNAME: Final = "hostname"
ATTR_VERSION: Final = "version"
ATTR_CPU_MODEL: Final = "cpu_model"
ATTR_CPU_CORES: Final = "cpu_cores"
ATTR_CPU_THREADS: Final = "cpu_threads"
ATTR_RAM_TOTAL: Final = "ram_total"
ATTR_SERVER_MODEL: Final = "server_model"
ATTR_BIOS_VERSION: Final = "bios_version"
ATTR_ARRAY_STATE: Final = "array_state"
ATTR_NUM_DISKS: Final = "num_disks"
ATTR_NUM_DATA_DISKS: Final = "num_data_disks"
ATTR_NUM_PARITY_DISKS: Final = "num_parity_disks"
ATTR_CONTAINER_ID: Final = "container_id"
ATTR_CONTAINER_IMAGE: Final = "container_image"
ATTR_CONTAINER_STATUS: Final = "container_status"
ATTR_VM_ID: Final = "vm_id"
ATTR_VM_VCPUS: Final = "vm_vcpus"
ATTR_VM_MEMORY: Final = "vm_memory"
ATTR_GPU_DRIVER_VERSION: Final = "gpu_driver_version"
ATTR_NETWORK_MAC: Final = "network_mac"
ATTR_NETWORK_IP: Final = "network_ip"
ATTR_NETWORK_SPEED: Final = "network_speed"

# Icons
ICON_CPU: Final = "mdi:cpu-64-bit"
ICON_MEMORY: Final = "mdi:memory"
ICON_TEMPERATURE: Final = "mdi:thermometer"
ICON_UPTIME: Final = "mdi:clock-outline"
ICON_ARRAY: Final = "mdi:harddisk"
ICON_PARITY: Final = "mdi:shield-check"
ICON_CONTAINER: Final = "mdi:docker"
ICON_VM: Final = "mdi:desktop-tower"
ICON_GPU: Final = "mdi:expansion-card"
ICON_NETWORK: Final = "mdi:ethernet"
ICON_UPS: Final = "mdi:battery"
ICON_POWER: Final = "mdi:power"
ICON_START: Final = "mdi:play"
ICON_STOP: Final = "mdi:stop"
ICON_RESTART: Final = "mdi:restart"

# Error messages
ERROR_CANNOT_CONNECT: Final = "cannot_connect"
ERROR_INVALID_AUTH: Final = "invalid_auth"
ERROR_UNKNOWN: Final = "unknown"
ERROR_TIMEOUT: Final = "timeout"
ERROR_ALREADY_CONFIGURED: Final = "already_configured"

# Service names
SERVICE_CONTAINER_START: Final = "container_start"
SERVICE_CONTAINER_STOP: Final = "container_stop"
SERVICE_CONTAINER_RESTART: Final = "container_restart"
SERVICE_VM_START: Final = "vm_start"
SERVICE_VM_STOP: Final = "vm_stop"
SERVICE_VM_RESTART: Final = "vm_restart"
SERVICE_ARRAY_START: Final = "array_start"
SERVICE_ARRAY_STOP: Final = "array_stop"
SERVICE_PARITY_CHECK_START: Final = "parity_check_start"
SERVICE_PARITY_CHECK_STOP: Final = "parity_check_stop"

