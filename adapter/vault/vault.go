package vault

import (
	"encoding/json"
	"fmt"
	"github.com/nkhoaa96/go-base/storage/local"
	"io"
	"net/http"
)

type Response struct {
	Name  string `json:"name"`
	Value struct {
		Raw                string `json:"raw"`
		Computed           string `json:"computed"`
		Note               string `json:"note"`
		RawVisibility      string `json:"rawVisibility"`
		ComputedVisibility string `json:"computedVisibility"`
	} `json:"value"`
	Success bool `json:"success"`
}

func GetSecretValue(key string) (string, error) {
	req, _ := http.NewRequest("GET", local.Getenv("KV_URL"), nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", local.Getenv("KV_TOKEN")))

	q := req.URL.Query()
	q.Add("project", local.Getenv("KV_PROJECT"))
	q.Add("config", local.Getenv("ENVIRONMENT"))
	q.Add("name", key)
	req.URL.RawQuery = q.Encode()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Cannot get ENV:", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var data Response
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("Cannot get ENV:", err)
	}
	return data.Value.Raw, nil
}
