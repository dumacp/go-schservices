package restclient

import (
	"fmt"
	"net/http"

	"github.com/dumacp/go-schservices/internal/utils"
)

func GetData(c *http.Client, id, urlin, filterHttpQuery string, sch, live bool) ([]byte, int, error) {
	if c == nil {
		return nil, 0, fmt.Errorf("client http empty")
	}

	filterHttp := fmt.Sprintf(filterHttpQuery, id, sch, live)
	url := fmt.Sprintf("%s%s", urlin, filterHttp)
	resp, code, err := utils.Get(c, url, "", "", nil)
	if err != nil {
		return nil, code, err
	}

	return resp, code, nil
}
