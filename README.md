# Creative Uploader

This is a web service built in [Go] that returns JSON-encoded data of image file uploads. The images are creative files (banners) in JPG,GIF or PNG format (SWF to be added later on). ZIP files (containing images) are supported as well.

##### Setup / run
 - Edit [creativeuploader.go] and edit the ``const bind`` value
 - Run: ``go run creativeuploader.go``

##### Usage
  - Upload the files to the web service, use the ``file`` parameter
  - Command line sample: ``curl http://www.atomx.com:5241/upload -i -F file=@~/image.gif``
  - Sample output:
```
{"files":[{"name":"image.gif","content":"PxNLUi/y......","filesize":12345,"width":300,"height":250,"mime":"image/gif"}],"error":""}
```
  - If the ``error`` field is not empty, there was an error during the upload
  - To return an HTML page containing the JSON response in JavaScript instead, POST to /upload?iframe

[Go]:https://www.golang.org/
[creativeuploader.go]:creativeuploader.go
