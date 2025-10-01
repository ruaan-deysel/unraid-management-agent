# Unraid Management Agent

A Go-based Unraid plugin that exposes comprehensive system monitoring and control via REST API and WebSockets, designed specifically for Home Assistant integration.

[![License](https://img.shields.io/badge/License-GPL%20v3-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)
[![Unraid](https://img.shields.io/badge/Unraid-6.12+-orange)](https://unraid.net/)

## Features

### 🔍 Monitoring
- **System Metrics**: CPU usage, RAM usage, temperatures, fan speeds, uptime
- **Array Status**: Array state, parity status, space usage
- **Disk Information**: Individual disk metrics, SMART data, temperatures
- **Docker Containers**: Container status, resource usage, ports
- **Virtual Machines**: VM status, resource allocation, state
- **UPS Status**: Battery level, load, runtime (if connected)
- **GPU Metrics**: NVIDIA GPU utilization and temperature (if available)
- **Shares**: User share space usage

### 🎛️ Control Operations
- **Docker**: Start, stop, restart, pause, unpause containers
- **VMs**: Start, stop, restart, pause, resume, hibernate, force-stop

### 🌐 API
- **REST API**: Full RESTful API with all monitoring and control endpoints
- **WebSocket**: Real-time event streaming for instant updates
- **CORS Enabled**: Works seamlessly with Home Assistant and web clients

## Installation

### Requirements
- Unraid 6.10 or later
- Go 1.23+ (for building from source)

### Via Community Applications (Recommended)
1. Open Unraid web UI
2. Go to **Apps** tab
3. Search for "Unraid Management Agent"
4. Click **Install**

### Manual Installation
```bash
# Download the plugin file
wget https://github.com/ruaandeysel/unraid-management-agent/releases/latest/download/unraid-management-agent.plg

# Install via Plugins tab
# Paste the URL in the "Install Plugin" field
```

### From Source
```bash
# Clone the repository
git clone https://github.com/ruaandeysel/unraid-management-agent.git
cd unraid-management-agent

# Build for Unraid (Linux/amd64)
make release

# Package the plugin
make package

# The plugin will be in build/unraid-management-agent-<version>.tgz
```

## Configuration

The plugin can be configured via the Unraid web UI or by editing the configuration file.

### Configuration File
Located at `/boot/config/plugins/unraid-management-agent/config.cfg`:

```ini
# Service configuration
SERVICE="enable"

# API configuration
PORT="8080"
ENABLE_CORS="yes"

# Feature toggles
ENABLE_UPS="yes"
ENABLE_GPU="yes"

# Collection intervals (seconds)
INTERVAL_SYSTEM="5"
INTERVAL_DISK="30"
INTERVAL_ARRAY="10"
INTERVAL_DOCKER="10"
INTERVAL_VM="10"
INTERVAL_UPS="10"
INTERVAL_SHARES="60"
```

## API Documentation

### Base URL
```
http://<unraid-ip>:8080/api/v1
```

### REST Endpoints

#### Monitoring (GET)
| Endpoint | Description |
|----------|-------------|
| `/health` | Health check |
| `/system` | System metrics (CPU, RAM, temps, uptime) |
| `/array` | Array status and parity info |
| `/disks` | All disk information |
| `/disks/{id}` | Single disk information |
| `/shares` | User share information |
| `/docker` | Docker container list |
| `/docker/{id}` | Single container info |
| `/vm` | Virtual machine list |
| `/vm/{id}` | Single VM info |
| `/ups` | UPS status |
| `/gpu` | GPU metrics |

#### Control (POST)
| Endpoint | Description |
|----------|-------------|
| `/docker/{id}/start` | Start container |
| `/docker/{id}/stop` | Stop container |
| `/docker/{id}/restart` | Restart container |
| `/docker/{id}/pause` | Pause container |
| `/docker/{id}/unpause` | Unpause container |
| `/vm/{id}/start` | Start VM |
| `/vm/{id}/stop` | Stop VM (graceful) |
| `/vm/{id}/restart` | Restart VM |
| `/vm/{id}/pause` | Pause VM |
| `/vm/{id}/resume` | Resume VM |
| `/vm/{id}/hibernate` | Hibernate VM |
| `/vm/{id}/force-stop` | Force stop VM |

### WebSocket
Connect to `ws://<unraid-ip>:8080/api/v1/ws` for real-time event updates.

**Event Format:**
```json
{
  "event": "system_update",
  "timestamp": "2025-10-01T04:00:00Z",
  "data": {
    // Event-specific payload
  }
}
```

## Home Assistant Integration

### REST Sensors
```yaml
# configuration.yaml
sensor:
  - platform: rest
    name: "Unraid CPU Usage"
    resource: "http://unraid-ip:8080/api/v1/system"
    value_template: "{{ value_json.cpu_usage_percent }}"
    unit_of_measurement: "%"
    scan_interval: 5
    
  - platform: rest
    name: "Unraid Array Status"
    resource: "http://unraid-ip:8080/api/v1/array"
    value_template: "{{ value_json.state }}"
    scan_interval: 10
```

### Control Services
```yaml
# configuration.yaml
rest_command:
  unraid_restart_container:
    url: "http://unraid-ip:8080/api/v1/docker/{{ container_id }}/restart"
    method: POST

# scripts.yaml
restart_plex:
  alias: "Restart Plex"
  sequence:
    - service: rest_command.unraid_restart_container
      data:
        container_id: "plex"
```

### Dashboard Card
```yaml
type: entities
title: Unraid System
entities:
  - entity: sensor.unraid_cpu_usage
    name: CPU Usage
  - entity: sensor.unraid_array_status
    name: Array Status
```

## Development

### Prerequisites
- Go 1.23+
- Make
- Git

### Building
```bash
# Install dependencies
make deps

# Build for current platform
make local

# Run tests
make test

# Build for Unraid (Linux/amd64)
make release

# Create plugin package
make package
```

### Mock Mode
For development on non-Unraid systems:
```bash
# Enable mock mode
export MOCK_MODE=true

# Run with mock data
./unraid-management-agent --mock
```

### Project Structure
```
unraid-management-agent/
├── main.go                    # Application entry point
├── daemon/
│   ├── cmd/                   # Commands
│   ├── common/                # Constants
│   ├── domain/                # Domain models
│   ├── dto/                   # Data transfer objects
│   ├── lib/                   # Utility libraries
│   ├── logger/                # Logging
│   └── services/
│       ├── api/               # HTTP/WebSocket server
│       ├── collectors/        # Data collectors
│       └── controllers/       # Control operations
├── meta/                      # Unraid plugin files
└── docs/                      # Documentation
```

## Troubleshooting

### Plugin Won't Start
1. Check logs: `/var/log/unraid-management-agent.log`
2. Verify port 8080 is not in use: `netstat -tlnp | grep 8080`
3. Restart the plugin from Unraid UI

### API Returns Empty Data
- Ensure you're running on Unraid (or use mock mode for testing)
- Check that required system commands are available
- Review logs for specific error messages

### WebSocket Connection Fails
- Verify CORS is enabled in configuration
- Check firewall settings
- Ensure WebSocket protocol is allowed by your client

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [ControlR](https://github.com/jbrodriguez/controlrd)
- Built with [Gorilla Mux](https://github.com/gorilla/mux) and [Gorilla WebSocket](https://github.com/gorilla/websocket)
- Designed for [Home Assistant](https://www.home-assistant.io/) integration

## Support

- **Issues**: [GitHub Issues](https://github.com/ruaandeysel/unraid-management-agent/issues)
- **Forum**: [Unraid Forums](https://forums.unraid.net/)
- **Discord**: Unraid Community Discord

## Roadmap

- [ ] Complete real data collection implementations
- [ ] Add authentication support
- [ ] MQTT integration
- [ ] Historical data storage
- [ ] Web UI dashboard
- [ ] Mobile app
- [ ] Prometheus exporter
- [ ] Grafana dashboard templates

---

**Made with ❤️ for the Unraid community**
