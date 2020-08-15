package reverse_proxy

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
	"strings"
)

type ReverseProxyOptions struct {
	TargetHost  string
	TargetPort  string
	RewritePath string
}

func ReverseProxyHandle(path string, option ReverseProxyOptions) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if strings.Index(ctx.Request.RequestURI, path) == 0 {
			client := &http.Client{}
			requestUrl := strings.Replace(ctx.Request.RequestURI, option.RewritePath, "", 1)
			url := option.TargetHost + ":" + option.TargetPort + "/" + requestUrl
			req, err := http.NewRequest(ctx.Request.Method, url, ctx.Request.Body)
			if err != nil {
				println(err)
				return
			}
			req.Header = ctx.Request.Header
			resp, err := client.Do(req)
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			for key, value := range resp.Header {
				if len(value) == 1 {
					ctx.Writer.Header().Add(key, value[0])
				}
			}
			ctx.Status(resp.StatusCode)
			_, err = ctx.Writer.Write(body)
			if err != nil {
				println(err)
			}
		} else {
			ctx.Next()
		}
	}
}
