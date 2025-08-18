# KleinPDF - A tiny PDF Compression Desktop App

A high-performance PDF compression desktop application built with Wails (Go + Web) and Preact frontend. macOS-only (Intel + Apple Silicon) with Ghostscript bundled inside the app.

## Screenshot

![Screenshot of PDF compressor app](./app-screenshot.jpg?raw=true "KleinPDF")

## 🚀 Features

- **📄 Advanced PDF Compression**: Compression using embedded Ghostscript
- **🖥️ Native Desktop Experience**: Built with Wails for seamless desktop integration
- **📁 Batch Processing**: Compress multiple PDF files simultaneously with concurrent processing
- **⚙️ Configurable Settings**: Multiple compression levels and advanced options
- **📊 Statistics Tracking**: Session and lifetime statistics for files compressed and data saved

## 📁 Project Structure

```
compressor/
├── main.go                     # Wails application entry point
├── wails.json                  # Wails configuration
├── go.mod                      # Go module dependencies
├── go.sum                      # Go module checksums
├── internal/                   # Go application code
│   ├── application/           # Core application logic
│   │   ├── app.go            # Main app struct with Wails bindings
│   │   ├── compression.go    # PDF compression handler with concurrent processing
│   │   ├── files.go          # File operations and download management
│   │   ├── preferences.go    # User preferences handler
│   │   ├── dialogs.go        # Native file/directory dialogs
│   │   ├── stats.go          # Statistics tracking and management
│   │   ├── types.go          # Application data structures
│   │   └── utils.go          # Utility functions
│   ├── binary/               # Embedded Ghostscript binary management
│   │   ├── generate.go       # Go generate script for downloading binaries
│   │   └── script.go         # Binary download and embedding logic
│   ├── config/               # Configuration management
│   │   └── config.go         # Application configuration
│   ├── database/             # Database initialization
│   │   └── database.go       # SQLite database setup
│   ├── models/               # Database models
│   │   └── preferences.go    # User preferences data model
│   └── services/             # Business logic services
│       ├── pdf.go            # PDF compression service using Ghostscript
│       └── preferences.go    # Preferences service
├── frontend/                 # Preact frontend
│   ├── src/                  # Source files
│   │   ├── main.tsx          # Frontend entry point
│   │   ├── app.tsx           # Main Preact component with UI
│   │   ├── styles.css        # Tailwind CSS styling
│   │   └── types/            # TypeScript type definitions
│   │       └── app.ts        # Frontend type definitions
│   ├── index.html            # HTML template
│   ├── package.json          # Frontend dependencies
│   ├── tsconfig.json         # TypeScript configuration
│   ├── vite.config.ts        # Vite build configuration
│   └── wailsjs/              # Auto-generated Wails bindings
└── build/                    # Built executables (generated)
```

## 🛠️ Technology Stack

- **Backend**: Go 1.25 with Wails v2 framework
- **Frontend**: Preact + TypeScript + Vite for fast, lightweight UI
- **Desktop Runtime**: Wails (native Go binaries)
- **PDF Compression**: Ghostscript 10.05.1 (embedded binary)
- **Database**: SQLite with GORM for data persistence
- **Styling**: Tailwind CSS with custom PDF-themed design
- **Build Tools**: Wails CLI, Vite, Go modules
- **Package Manager**: pnpm for frontend dependencies

## ⚡ Performance Features

- **Concurrent Processing**: Multi-threaded compression (up to 8 cores)
- **Direct File Processing**: No temporary file copying - Ghostscript reads original and writes compressed directly
- **Bundle Size**: Small native binary with embedded resources
- **Startup Time**: Native binary execution with minimal overhead

## 🚀 Quick Start

### Prerequisites (macOS)

- Go 1.25 or later
- Node.js 19+
- pnpm (recommended) or npm
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Installation

1. **Clone the repository**:

   ```bash
   git clone <repository-url>
   cd compressor
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




