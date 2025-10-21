# Contributing to Sentinel

Thank you for your interest in contributing to Sentinel! This guide will help you get started.

## Development Setup

### Prerequisites

Ensure you have all the required tools installed:
- Linux kernel 5.8+
- clang and llvm
- libbpf development files
- Go 1.20+
- Node.js 18+

### Clone and Build

```bash
git clone https://github.com/your-org/sentinel.git
cd sentinel

# Build all components
./scripts/build-all.sh
```

### Running in Development Mode

```bash
# Terminal 1: Start backend
cd sentinel/backend
make dev

# Terminal 2: Start frontend
cd sentinel/frontend
npm start

# Terminal 3: Start agent (requires root)
cd agent
sudo make dev
```

Or use the convenience script:
```bash
./scripts/run-dev.sh
```

## Project Structure

```
.
├── agent/              # eBPF monitoring agent
│   ├── ebpf/          # eBPF C programs
│   └── src/           # Go userspace program
├── sentinel/          # Centralized server
│   ├── backend/       # Go backend API
│   └── frontend/      # React frontend
├── docs/              # Documentation
└── scripts/           # Build/deployment scripts
```

## Making Changes

### Agent Development

#### Adding New eBPF Programs

1. Create your eBPF program in `agent/ebpf/`:
```c
// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

SEC("kprobe/your_function")
int trace_your_function(struct pt_regs *ctx) {
    // Your tracing code
    return 0;
}

char LICENSE[] SEC("license") = "GPL";
```

2. Add collector in `agent/src/pkg/collector/`:
```go
type YourCollector struct {
    // Collector fields
}

func NewYourCollector() (*YourCollector, error) {
    // Initialize collector
}
```

3. Update Makefile to compile new eBPF program
4. Integrate collector in `agent/src/main.go`

#### Testing eBPF Programs

```bash
cd agent
make clean
make ebpf

# Verify eBPF objects
ls -lh build/*.bpf.o

# Test loading
sudo bpftool prog load build/your_program.bpf.o /sys/fs/bpf/test
```

### Backend Development

#### Adding New API Endpoints

1. Define handler in `sentinel/backend/pkg/api/server.go`:
```go
func (s *Server) YourHandler(w http.ResponseWriter, r *http.Request) {
    // Handler implementation
}
```

2. Register route in `sentinel/backend/main.go`:
```go
router.HandleFunc("/api/v1/your-endpoint", apiServer.YourHandler).Methods("GET")
```

3. Add storage method if needed in `sentinel/backend/pkg/storage/memory.go`

#### Adding Storage Backends

1. Create new storage implementation:
```go
type PostgresStore struct {
    // Postgres-specific fields
}

func (s *PostgresStore) AddEvents(data []byte) error {
    // Implementation
}
```

2. Implement the Store interface
3. Add configuration options
4. Update main.go to support new backend

### Frontend Development

#### Adding New Components

1. Create component in `sentinel/frontend/src/components/`:
```jsx
import React from 'react';
import './YourComponent.css';

const YourComponent = () => {
    return (
        <div className="your-component">
            {/* Component content */}
        </div>
    );
};

export default YourComponent;
```

2. Create corresponding CSS file
3. Import and use in App.js or other components

#### Adding API Calls

1. Add API function in `sentinel/frontend/src/services/api.js`:
```javascript
export const fetchYourData = async () => {
    const response = await api.get('/your-endpoint');
    return response.data;
};
```

2. Use in components with useEffect or event handlers

## Code Style

### Go
- Follow standard Go formatting (use `gofmt`)
- Add comments for exported functions
- Use meaningful variable names
- Handle all errors explicitly

### C (eBPF)
- Follow Linux kernel coding style
- Use BPF CO-RE for portability
- Add SPDX license identifier
- Keep programs simple and efficient

### JavaScript/React
- Use functional components with hooks
- Follow ESLint rules
- Use meaningful component and variable names
- Add PropTypes or TypeScript for type safety

## Testing

### Agent Tests
```bash
cd agent/src
go test ./...
```

### Backend Tests
```bash
cd sentinel/backend
go test ./...
```

### Frontend Tests
```bash
cd sentinel/frontend
npm test
```

## Documentation

When adding new features:

1. Update relevant documentation in `docs/`
2. Add code comments
3. Update README.md if needed
4. Add examples to USER_GUIDE.md

## Commit Guidelines

### Commit Message Format
```
type(scope): subject

body

footer
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting)
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Build/tooling changes

**Examples:**
```
feat(agent): add disk I/O monitoring

Implements eBPF programs to monitor disk read/write operations
including latency tracking and per-process I/O statistics.

Closes #123
```

```
fix(backend): resolve WebSocket connection leak

Properly close WebSocket connections when clients disconnect
to prevent resource exhaustion.

Fixes #456
```

## Pull Request Process

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Make** your changes
4. **Test** your changes thoroughly
5. **Commit** with descriptive messages
6. **Push** to your fork
7. **Create** a Pull Request

### PR Checklist
- [ ] Code follows project style guidelines
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] All tests passing
- [ ] No merge conflicts

## Performance Considerations

### eBPF Programs
- Minimize operations in hot paths
- Use ring buffers for large data
- Avoid unbounded loops
- Use BPF maps efficiently

### Backend
- Batch database operations
- Use connection pooling
- Implement rate limiting
- Cache frequently accessed data

### Frontend
- Lazy load components
- Implement virtual scrolling for large lists
- Debounce API calls
- Optimize re-renders with React.memo

## Security

### Reporting Vulnerabilities
Please report security issues to security@example.com, not in public issues.

### Security Best Practices
- Never commit secrets or credentials
- Validate all input data
- Use prepared statements for SQL
- Implement proper authentication
- Enable HTTPS/WSS in production

## Getting Help

- **Questions**: Open a GitHub discussion
- **Bugs**: Create a GitHub issue
- **Chat**: Join our Slack/Discord channel

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Recognition

Contributors will be added to CONTRIBUTORS.md. Thank you for making Sentinel better!
