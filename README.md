# Make Small PDF - Wails + Preact Desktop App

A high-performance PDF compression desktop application built with Wails (Go + Web) and Preact frontend. Migrated from Electron to Wails for massive performance improvements while maintaining the exact same functionality and design.

## Screenshot

![Screenshot of PDF compressor app](./app-screenshot.jpg?raw=true "Make Small PDF")

## 🚀 Features

- **📄 PDF Compression**: Advanced compression using Ghostscript
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
├── wails.json              # Wails configuration
├── go.mod                  # Go module dependencies
├── internal/               # Go application code
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
├── bundled/               # Bundled Ghostscript
└── build/                 # Built executables
```

## 🛠️ Technology Stack

- **Backend**: Go 1.23+ with Wails v2
- **Frontend**: Preact + Vite for fast, lightweight UI
- **Desktop Runtime**: Wails (native Go binaries)
- **PDF Compression**: Ghostscript (bundled)
- **Database**: SQLite with GORM
- **Build Tools**: Wails CLI, Vite, Go modules

## ⚡ Performance Benefits

**Wails vs Electron Comparison:**

- **Bundle Size**: ~8MB vs ~140MB (94% reduction)
- **Memory Usage**: ~80% less RAM consumption
- **Startup Time**: Native binary execution vs V8 interpretation
- **Frontend Size**: Preact (~3KB) vs React (~42KB gzipped)

## 🚀 Quick Start

### Prerequisites

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

## 🚀 Production Deployment

**Build for production**:

```bash
wails build
```

This creates platform-specific executables in the `build/` directory.

## 📦 Bundling Ghostscript

This app requires a bundled Ghostscript distribution. Use the Homebrew-based bundler to prepare the necessary files:

### Homebrew-based bundling

Requires macOS with Homebrew installed. The bundler will install Ghostscript via Homebrew (if not already installed) and copy the required files into `./bundled/ghostscript`:

```bash
go run ./script/bundler.go
```

This will place the following:

- `bundled/ghostscript/bin/gs`
- `bundled/ghostscript/lib/*` (dynamic libraries)
- `bundled/ghostscript/share/ghostscript/<version>/*` (resources)

The application will only use the bundled Ghostscript. System-installed versions are ignored by design.

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

Edit `wails.json` to customize:

- Application metadata (name, version, author)
- Build settings and output paths
- Platform-specific configurations
- Frontend framework settings

## 📝 Migration Notes

See `migration-to-wails.md` for detailed information about the migration from Electron to Wails, including performance comparisons and technical decisions.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE.txt](LICENSE.txt) file for details.
