# Pandoras Box

[![GoDoc](https://godoc.org/github.com/capnspacehook/pandorasbox?status.svg)](https://godoc.org/github.com/capnspacehook/pandorasbox)

`pandorasbox` is a Go package that allows for simple use of both a host's filesystem, and a virtual filesystem.

The design goal of Pandora's Box is to easily facilitate the use of a transparently-encrypted VFS (virtual filesystem), and the host's filesystem. It does this by providing functions and methods that operate and look the same as the Go standard library `os` package. If you want to interact with the VFS, pass in a path that starts with `vfs://`, and Pandora's Box will automatically use the VFS. Otherwise, the host's filesystem will be used.

## Using Pandora's Box

Because Pandora's Box has the same interface as the `os` package, giving your code access to a VFS is often as easy as importing `pandorasbox` and replacing `os` calls to `box` calls. Take this super simple function that copies files: 

```go
import "os"

func CopyFile(srcFile, dstFile string) error {
    out, err := os.Create(dstFile)
    defer out.Close()
    if err != nil {
      return err
    }

    in, err := os.Open(srcFile)
    defer in.Close()
    if err != nil {
    return err
    }

    _, err = io.Copy(out, in)
    if err != nil {
    return err
    }

    return nil
}
```

All it takes to make this function VFS-friendly is switching from using `os` to `pandorasbox`:

```go
import box "github.com/capnspacehook/pandorasbox"

func init() {
    box.InitGlobalBox()
}

func CopyFile(srcFile, dstFile string) error {
    out, err := box.Create(dstFile)
    if err != nil {
      return err
    }
    defer out.Close()

    in, err := box.Open(srcFile)
    if err != nil {
      return err
    }
    defer in.Close()

    _, err = io.Copy(out, in)
    if err != nil {
      return err
    }

    return nil
}
```

### Global vs. Local VFS

You probably noticed the call to `box.InitGlobalBox()` in the last example. This has to be called **before** the global VFS can be used. 
For ease of use, Pandora's box provides a global `Box` that is easily accessible, but in some cases a local `Box` may be desired. If you don't wish to use the global `Box`, don't call `box.InitGlobalBox()`, instead create a locally scoped `Box` by calling `box.NewBox()`. This allows you to easily pass a `Box` into functions or methods or embed a `Box` in a struct.

### `io/ioutil` and `path/filepath` Functions

Pandora's Box also provides helper functions that are identical to functions from `io/ioutil` and `path/filepath`. These should be used of the Go standard library packages when using a `Box`. The Pandora's Box versions are VFS-friendly, and will work seamlessly with a VFS, while the Go standard library packages will not. If you're using the global `Box`, the `io/ioutil` functions can be called from the main import: `github.com/capnspacehook/pandorasbox`. If you're using a local `Box`, you'll need to import `github.com/capnspacehook/pandorasbox/ioutil` and pass in your `Box` to those functions.

Example (error handling omitted):

```go 
import (
    box "github.com/capnspacehook/pandorasbox"
    "github.com/capnspacehook/pandorasbox/ioutil"
)

func init() {
    box.InitGlobalBox()
}

func WriteFileGlobalBox() {
    box.WriteFile("vfs://file.txt", []byte("Testing testing 1 2 3"), 0644)
    data, _ := box.ReadFile("vfs://file.txt")
    fmt.Println(string(data))
}

func WriteFileLocalBox() {
    myBox := box.NewBox()

    ioutil.WriteFile(myBox, "vfs://file.txt", []byte("Testing testing 1 2 3"), 0644)
    data, _ := ioutil.ReadFile(myBox, "vfs://file.txt")
    fmt.Println(string(data))
}
```

### Forcing use of Host FS/VFS

If for some reason you need to force the usage of either the host's filesystem or the VFS, Pandora's box has you covered. All of `pandorasbox`'s functions that are in also in `os` have 3 variants: normal, OS, and VFS. The normal variant auto-detirmines what to use based off the input path, as described earlier. The OS and VFS variants force the usage of a specific filesystem. For instance, `pandorasbox.Mkdir()` will auto-detirmine which filesystem to use, while `pandorasbox.OSMkdir()` will always use the host's filesystem, and `pandorasbox.VFSMkdir()` will always use the VFS. 

### Memory Safety

All files in the VFS are encrypted when not in use. When files from the VFS are opened, they are decrypted for the duration of the call that opened them. VFS files are then re-encrypted with a different random key when reading or writing from them is finished. That is, files in the VFS are only decrypted in memory for a brief time while the underlying data needs to be accessed. In other words, calling `Open()` on a VFS file **will not** decrypt it until `Close()` is called on it. It will only be decrypted in memory when it is internally opened by methods like `Read()`, `Write()`, `Truncate()`, etc. And it is immediately closed afterwards. So opening a VFS file and calling `Read()` on it 3 times will decrypt and re-encrypt it 3 times. This is to make sure data is encrypted in memory whenever possible.

For more information about the exact cryptographic code and algorithms used, refer to this repo: https://github.com/awnumar/memguard.

## Acknowledgements

Thanks to AbsFs contributors for the amazing repos, 70% of the code is from repos from [this organization](https://github.com/absfs).

Took some VFS specific tests from [this repo](https://github.com/blang/vfs), thanks to [blang](https://github.com/blang) for some good VFS tests.

Thanks to [awnumar](https://github.com/awnumar) for [memguard](https://github.com/awnumar/memguard), he created a great repo that is very easy to use safely.
