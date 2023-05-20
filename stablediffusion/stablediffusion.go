package stablediffusion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func buildURL(baseUrl string, path string) (*url.URL, error) {
	u, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	u.Path = path

	return u, nil
}

// SimpleImageRequest is all of the parameters needed to generate an image.
type SimpleImageRequest struct {
	Prompt           string   `json:"prompt"`
	NegativePrompt   string   `json:"negative_prompt"`
	Styles           []string `json:"styles"`
	Seed             int      `json:"seed"`
	SamplerName      string   `json:"sampler_name"`
	BatchSize        int      `json:"batch_size"`
	NIter            int      `json:"n_iter"`
	Steps            int      `json:"steps"`
	CfgScale         int      `json:"cfg_scale"`
	Width            int      `json:"width"`
	Height           int      `json:"height"`
	SNoise           int      `json:"s_noise"`
	EnableHr         bool     `json:"enable_hr"`
	HrScale          int      `json:"hr_scale"`
	HrUpscaler       string   `json:"hr_upscaler"`
	OverrideSettings struct {
	} `json:"override_settings"`
	OverrideSettingsRestoreAfterwards bool `json:"override_settings_restore_afterwards"`
	SaveImages                        bool `json:"save_images"`
}

type ImageResponse struct {
	Images [][]byte `json:"images"`
	Info   string   `json:"info"`
}

type Algorithm struct {
	Name    string            `json:"name"`
	Aliases []string          `json:"aliases"`
	Options map[string]string `json:"options"`
}

type Upscaler struct {
	Name      string `json:"name"`
	ModelName string `json:"model_name"`
	ModelPath string `json:"model_path"`
	ModelUrl  string `json:"model_url"`
	scale     int    `json:"scale"`
}

type ImageInfo struct {
	Prompt                string      `json:"prompt"`
	AllPrompts            []string    `json:"all_prompts"`
	NegativePrompt        string      `json:"negative_prompt"`
	AllNegativePrompts    []string    `json:"all_negative_prompts"`
	Seed                  int         `json:"seed"`
	AllSeeds              []int       `json:"all_seeds"`
	Subseed               int         `json:"subseed"`
	AllSubseeds           []int       `json:"all_subseeds"`
	SubseedStrength       int         `json:"subseed_strength"`
	Width                 int         `json:"width"`
	Height                int         `json:"height"`
	SamplerName           string      `json:"sampler_name"`
	CfgScale              float64     `json:"cfg_scale"`
	Steps                 int         `json:"steps"`
	BatchSize             int         `json:"batch_size"`
	RestoreFaces          bool        `json:"restore_faces"`
	FaceRestorationModel  interface{} `json:"face_restoration_model"`
	SdModelHash           string      `json:"sd_model_hash"`
	SeedResizeFromW       int         `json:"seed_resize_from_w"`
	SeedResizeFromH       int         `json:"seed_resize_from_h"`
	DenoisingStrength     int         `json:"denoising_strength"`
	ExtraGenerationParams struct {
	} `json:"extra_generation_params"`
	IndexOfFirstImage             int           `json:"index_of_first_image"`
	Infotexts                     []string      `json:"infotexts"`
	Styles                        []interface{} `json:"styles"`
	JobTimestamp                  string        `json:"job_timestamp"`
	ClipSkip                      int           `json:"clip_skip"`
	IsUsingInpaintingConditioning bool          `json:"is_using_inpainting_conditioning"`
}

var (
	Default *Client = &Client{
		HTTP: http.DefaultClient,
	}
)

func Generate(sdUrl string, ctx context.Context, inp SimpleImageRequest) (*ImageResponse, error) {
	return Default.Generate(sdUrl, ctx, inp)
}

type Client struct {
	HTTP *http.Client
}

func (c *Client) Generate(sdUrl string, ctx context.Context, inp SimpleImageRequest) (*ImageResponse, error) {
	u, err := buildURL(sdUrl, "/sdapi/v1/txt2img")
	if err != nil {
		return nil, fmt.Errorf("error building URL: %w", err)
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(inp); err != nil {
		return nil, fmt.Errorf("error encoding json: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error status code: %w", err)
	}

	var result ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error parsing ImageResponse: %w", err)
	}

	return &result, nil
}

func GetSamplers(sdUrl string, ctx context.Context) (map[string]Algorithm, error) {
	u, err := buildURL(sdUrl, "/sdapi/v1/samplers")
	if err != nil {
		return nil, fmt.Errorf("error building URL: %w", err)
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(""); err != nil {
		return nil, fmt.Errorf("error parsing SamplerResponse: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), &buf)
	resp, err := Default.HTTP.Do(req)
	samplers := []Algorithm{}
	if err := json.NewDecoder(resp.Body).Decode(&samplers); err != nil {
		return nil, fmt.Errorf("error parsing ImageResponse: %w", err)
	}
	response := make(map[string]Algorithm)
	for _, sampler := range samplers {
		response[sampler.Name] = sampler
	}

	if err != nil {
		return nil, fmt.Errorf("error fetching response: %w", err)
	}
	return response, nil
}

func GetUpscalers(sdUrl string, ctx context.Context) (map[string]Upscaler, error) {
	u, err := buildURL(sdUrl, "/sdapi/v1/upscalers")
	if err != nil {
		return nil, fmt.Errorf("error building URL: %w", err)
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(""); err != nil {
		return nil, fmt.Errorf("error parsing SamplerResponse: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), &buf)
	resp, err := Default.HTTP.Do(req)
	upscalers := []Upscaler{}
	if err := json.NewDecoder(resp.Body).Decode(&upscalers); err != nil {
		return nil, fmt.Errorf("error parsing ImageResponse: %w", err)
	}
	response := make(map[string]Upscaler)
	for _, upscaler := range upscalers {
		response[upscaler.Name] = upscaler
	}

	if err != nil {
		return nil, fmt.Errorf("error fetching response: %w", err)
	}
	return response, nil
}
