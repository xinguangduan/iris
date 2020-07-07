package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/basicauth"
)

func init() {
	os.Mkdir("./uploads", 0700)
}

const (
	maxSize   = 1 * iris.GB
	uploadDir = "./uploads"
)

func main() {
	app := iris.New()

	view := iris.HTML("./views", ".html")
	view.AddFunc("formatBytes", func(b int64) string {
		const unit = 1000
		if b < unit {
			return fmt.Sprintf("%d B", b)
		}
		div, exp := int64(unit), 0
		for n := b / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %cB",
			float64(b)/float64(div), "kMGTPE"[exp])
	})
	app.RegisterView(view)

	// Serve assets (e.g. javascript, css).
	// app.HandleDir("/public", "./public")

	app.Get("/", index)

	app.Get("/upload", uploadView)
	app.Post("/upload", upload)

	filesRouter := app.Party("/files")
	{
		filesRouter.HandleDir("/", uploadDir, iris.DirOptions{
			Gzip:     true,
			ShowList: true,
			DirList: iris.DirListRich(iris.DirListRichOptions{
				// Optionally, use a custom template for listing:
				// Tmpl: dirListRichTemplate,
				TmplName: "dirlist.html",
			}),
		})

		auth := basicauth.New(basicauth.Config{
			Users: map[string]string{
				"myusername": "mypassword",
			},
		})

		filesRouter.Delete("/{file:path}", auth, deleteFile)
	}

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.Redirect("/upload")
}

func uploadView(ctx iris.Context) {
	now := time.Now().Unix()
	h := md5.New()
	io.WriteString(h, strconv.FormatInt(now, 10))
	token := fmt.Sprintf("%x", h.Sum(nil))

	ctx.View("upload.html", token)
}

func upload(ctx iris.Context) {
	ctx.SetMaxRequestBodySize(maxSize)

	_, err := ctx.UploadFormFiles(uploadDir, beforeSave)
	if err != nil {
		ctx.StopWithError(iris.StatusPayloadTooRage, err)
		return
	}

	ctx.Redirect("/files")
}

func beforeSave(ctx iris.Context, file *multipart.FileHeader) {
	ip := ctx.RemoteAddr()
	ip = strings.ReplaceAll(ip, ".", "_")
	ip = strings.ReplaceAll(ip, ":", "_")

	file.Filename = ip + "-" + file.Filename
}

func deleteFile(ctx iris.Context) {
	// It does not contain the system path,
	// as we are not exposing it to the user.
	fileName := ctx.Params().Get("file")

	filePath := path.Join(uploadDir, fileName)

	if err := os.RemoveAll(filePath); err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	ctx.Redirect("/files")
}