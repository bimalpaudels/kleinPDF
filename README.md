# Make Small PDF - Wails + Preact Desktop App

A high-performance PDF compression desktop application built with Wails (Go + Web) and Preact frontend. macOS-only (Intel + Apple Silicon) with Ghostscript bundled inside the app.

## Screenshot

![Screenshot of PDF compressor app](./app-screenshot.jpg?raw=true "Make Small PDF")

## 🚀 Features

- **📄 PDF Compression**: Advanced compression using Ghostscript (embedded)
- **🖥️ Desktop App**: Native desktop experience with Wails
- **📁 Multiple Files**: Compress multiple PDF files at once
- **⚡ Fast Processing**: Efficient batch processing with Go backend
- **🎯 Auto-Download**: Automatic file saving to preferred folder
- **⚙️ Preferences**: Configurable settings and download folder
- **🧹 Auto-Cleanup**: Automatic temporary file cleanup

## 📁 Project Structure

```
pdf-compressor/
├── main.go                 # Wails application entry point
├── app.go                  # Main app struct with methods
├── generate.go             # Go generate script for downloading binaries
├── wails.json              # Wails configuration
├── go.mod                  # Go module dependencies
├── internal/               # Go application code
│   ├── binary/            # Embedded Ghostscript binary
│   ├── config/            # Configuration management
│   ├── database/          # Database initialization
│   ├── models/            # Database models
│   └── services/          # Business logic services
├── frontend/              # Preact frontend
│   ├── src/               # Source files
│   │   ├── main.jsx       # Frontend entry point
│   │   ├── app.jsx        # Main Preact component
│   │   └── app.css        # Styling
│   ├── dist/              # Built frontend assets
│   └── wailsjs/           # Auto-generated bindings
└── build/                 # Built executables
```

## 🛠️ Technology Stack

- **Backend**: Go 1.23+ with Wails v2
- **Frontend**: Preact + Vite for fast, lightweight UI
- **Desktop Runtime**: Wails (native Go binaries)
- **PDF Compression**: Ghostscript (embedded binary)
- **Database**: SQLite with GORM
- **Build Tools**: Wails CLI, Vite, Go modules

## ⚡ Performance Benefits

- **Bundle Size**: Small native binary with embedded resources
- **Memory Usage**: Low RAM consumption
- **Startup Time**: Native binary execution

## 🚀 Quick Start

### Prerequisites (macOS)

- Go 1.23 or later
- Node.js 19+
- pnpm (recommended) or npm
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Installation

1. **Clone the repository**:

   ```bash
   git clone <repository-url>
   cd pdf-compressor
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   cd frontend && pnpm install && cd ..
   ```

### Development

**Start the application in development mode**:

```bash
wails dev
```

This will:

- Start the Go backend integrated with Wails
- Launch the native application with hot reload
- Enable frontend development tools

**Generate bindings after Go changes**:

```bash
wails generate module
```

## 🚀 Production Build

```bash
wails build
```

This creates a macOS app in the `build/` directory.

## 📦 Ghostscript Binary Management

This app uses architecture-specific Ghostscript binaries directly embedded from [GitHub releases](https://github.com/bimalpaudels/kleinPDF-ghostscript-binary/releases). The binary is automatically downloaded and embedded during build time using Go's `go:generate` feature.

### Binary Generation

The appropriate binary for your system architecture is automatically downloaded during build:

```bash
# Generate the binary (happens automatically during build)
go generate ./internal/binary
```

Supported architectures:
- **Apple Silicon** (arm64): `ghostscript-10.05.1-macos-arm64`
- **Intel Macs** (amd64): `ghostscript-10.05.1-macos-x86_64`

The binary is embedded directly into the application using Go's `embed` package, eliminating the need for complex archive extraction.

System-installed Ghostscript is not used.

## 📦 Package Management

This project uses **pnpm** for frontend dependency management:

- **Faster**: Parallel installation and efficient disk usage
- **Reliable**: Deterministic dependency resolution
- **Efficient**: Shared dependencies across projects

### Frontend Development

```bash
cd frontend
pnpm install          # Install dependencies
pnpm dev             # Start development server
pnpm build           # Build for production
```

## 🔧 Configuration

Edit `wails.json` to customize basic app metadata and frontend build commands.

## 🤝 Contributing

- macOS-only support at this time (Intel + Apple Silicon)
- PRs that simplify macOS support are welcome

## 📄 License

This project is licensed under the MIT License - see the [LICENSE.txt](LICENSE.txt) file for details.
