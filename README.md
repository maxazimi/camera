# Camera Package

This package is designed for Go development with CGo and provides support for macOS, Linux, Windows, and Android.

## Features

- **Cross-Platform**: Supports macOS, Linux, Windows, and Android.
- **CGo Integration**: Utilizes CGo for efficient low-level operations.
- **Modular Design**: Easy to integrate and extend for various camera functionalities.

## Dependencies

- **macOS**: Requires Xcode Command Line Tools.
- **Linux**: Requires v4l2 (Video for Linux 2).
- **Windows**: Requires MinGW or MSYS2.
- **Android**: Requires NDK 24.

## iOS Support

iOS support is not yet included. Contributions are highly appreciated.

## TODO

- [ ] **Fix Bugs**: Ongoing effort to identify and fix bugs.
- [ ] **Add Front Camera Support for Android**: Currently, only the rear camera is supported.
- [ ] **Fix Logging in C Code**: Improve logging functionality on the C side.

## License

This project is licensed under the MIT License.

## Credits

Special thanks to [Kosua20](https://github.com/kosua20/sr_webcam) for their contributions to the macOS implementation.

## Related Projects

Check out [my QR repository](https://github.com/maxazimi/qr) that uses this package for QR scan demonstrations.
