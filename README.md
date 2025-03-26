# port-scanner

## Usage
## Compilation

1. Open a terminal/command prompt
2. Navigate to the directory containing `portscanner.go`
3. Run:
```
go build portscanner.go
```
This creates an executable named `portscanner` (or `portscanner.exe` on Windows)
Run the scanner with default settings:
```
./portscanner
```

Or customize the scan with flags:
```
./portscanner -start=192.168.1.1 -end=192.168.1.255 -ports=22,80,443,8080 -timeout=1s -concurrent=100
```

## Command Line Flags

- `-start`: Starting IP address (default: "192.168.1.1")
- `-end`: Ending IP address (default: "192.168.1.10")
- `-ports`: Comma-separated list of ports to scan (default: "22,80,443")
- `-timeout`: Connection timeout duration (default: "2s")
  - Examples: "500ms" (milliseconds), "2s" (seconds)
- `-concurrent`: Maximum number of concurrent scans (default: 50)

## Examples

Scan a single IP with specific ports:
```
./portscanner -start=192.168.1.1 -end=192.168.1.1 -ports=22,80,443
```

Scan a subnet with common ports:
```
./portscanner -start=192.168.1.0 -end=192.168.1.255 -ports=80,443,3389
```

Fast scan with more concurrent connections:
```
./portscanner -start=10.0.0.1 -end=10.0.0.255 -timeout=500ms -concurrent=200
```

## Notes

- Ports must be between 1 and 65535
- Requires network connectivity
- May require root/admin privileges on some systems
- Use responsibly and only on networks you have permission to scan
- Ctrl+C to stop the scan

## Output

- Shows open ports as they're found: "Port [number] is open on [IP]"
- Shows errors if they occur: "Error scanning [IP]:[port] - [error message]"
- Silent for closed ports to reduce output noise

## Troubleshooting

### Common Output Examples

When running the port scanner, you might see output like this:
```
Error scanning 192.168.1.2:22 - dial tcp 192.168.1.2:22: i/o timeout
Error scanning 192.168.1.7:80 - dial tcp 192.168.1.7:80: i/o timeout
Error scanning 192.168.1.1:22 - dial tcp 192.168.1.1:22: i/o timeout
```
This indicates that the scanner couldn't connect to those IP:port combinations within the timeout period.

### Possible Causes and Solutions

- **Closed Ports**: The ports might be closed on the target systems
  - Solution: Verify if services are running on those ports
- **Unreachable IPs**: The IP addresses might not be active on your network
  - Solution: Ping the IPs to check reachability (`ping 192.168.1.1`)
- **Firewall Blocking**: A firewall might be blocking the connection attempts
  - Solution: Check firewall settings on both your machine and the target
- **Timeout Too Short**: The default 2-second timeout might be insufficient
  - Solution: Increase the timeout (e.g., `-timeout=5s`)
- **Permission Issues**: You might need elevated privileges
  - Solution: Run with sudo/admin rights (e.g., `sudo ./portscanner`)

### Tips
- Test with known open ports first (e.g., a local web server on port 80)
- Reduce the IP range or number of ports if scanning takes too long
- Use `-concurrent` wisely to avoid overwhelming your network