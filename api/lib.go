package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
)

// ImageArchitecture defines model for ImageArchitecture.
type ImageArchitecture int

// Defines values for ImageArchitecture.
const (
	Arm64 ImageArchitecture = iota
	X8664
)

func (v *ImageArchitecture) UnmarshalJSON(buf []byte) error {
	switch strings.Trim(string(buf), "\"") {
	case "arm64":
		*v = Arm64
	case "x86_64":
		*v = X8664
	default:
		return fmt.Errorf("failed to unmarshal json for ImageArchitecture: %s", string(buf))
	}
	return nil
}

func (v *ImageArchitecture) String() string {
	switch *v {

	case Arm64:
		return "arm64"
	case X8664:
		return "x86_64"
	default:
		return ""
	}
}

// InstanceActionUnavailableCode defines model for InstanceActionUnavailableCode.
type InstanceActionUnavailableCode int

// Defines values for InstanceActionUnavailableCode.
const (
	VmHasNotLaunched InstanceActionUnavailableCode = iota
	VmIsTerminating
	VmIsTooOld
)

func (v *InstanceActionUnavailableCode) UnmarshalJSON(buf []byte) error {
	switch strings.Trim(string(buf), "\"") {
	case "vm-has-not-launched":
		*v = VmHasNotLaunched
	case "vm-is-terminating":
		*v = VmIsTerminating
	case "vm-is-too-old":
		*v = VmIsTooOld
	default:
		return fmt.Errorf("failed to unmarshal json for InstanceActionUnavailableCode: %s", string(buf))
	}
	return nil
}

func (v *InstanceActionUnavailableCode) String() string {
	switch *v {
	case VmHasNotLaunched:
		return "vm-has-not-launched"
	case VmIsTerminating:
		return "vm-is-terminating"
	case VmIsTooOld:
		return "vm-is-too-old"
	default:
		return ""
	}
}

// Status - The current status of the instance.
type Status int

// Defines values for Status.
const (
	StatusActive Status = iota
	StatusBooting
	StatusTerminated
	StatusTerminating
	StatusUnhealthy
)

func (v *Status) UnmarshalJSON(buf []byte) error {
	switch strings.Trim(string(buf), "\"") {
	case "active":
		*v = StatusActive
	case "booting":
		*v = StatusBooting
	case "terminated":
		*v = StatusTerminated
	case "terminating":
		*v = StatusTerminating
	case "unhealthy":
		*v = StatusUnhealthy
	default:
		return fmt.Errorf("failed to unmarshal json for Status: %s", string(buf))
	}
	return nil
}

func (v Status) String() string {
	switch v {
	case StatusActive:
		return "active"
	case StatusBooting:
		return "booting"
	case StatusTerminated:
		return "terminated"
	case StatusTerminating:
		return "terminating"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return ""
	}
}

// Region defines model for PublicRegionCode.
type Region int

// Defines values for Region.
const (
	AsiaNortheast1 Region = iota
	AsiaNortheast2
	AsiaSouth1
	AustraliaEast1
	EuropeCentral1
	MeWest1
	TestEast1
	TestWest1
	UsEast1
	UsEast2
	UsEast3
	UsMidwest1
	UsMidwest2
	UsSouth1
	UsSouth2
	UsSouth3
	UsWest1
	UsWest2
	UsWest3
)

func ParseRegion(s string) (r Region, err error) {
	switch s {
	case "asia-northeast-1":
		r = AsiaNortheast1
	case "asia-northeast-2":
		r = AsiaNortheast2
	case "asia-south-1":
		r = AsiaSouth1
	case "australia-east-1":
		r = AustraliaEast1
	case "europe-central-1":
		r = EuropeCentral1
	case "me-west-1":
		r = MeWest1
	case "test-east-1":
		r = TestEast1
	case "test-west-1":
		r = TestWest1
	case "us-east-1":
		r = UsEast1
	case "us-east-2":
		r = UsEast2
	case "us-east-3":
		r = UsEast3
	case "us-midwest-1":
		r = UsMidwest1
	case "us-midwest-2":
		r = UsMidwest2
	case "us-south-1":
		r = UsSouth1
	case "us-south-2":
		r = UsSouth2
	case "us-south-3":
		r = UsSouth3
	case "us-west-1":
		r = UsWest1
	case "us-west-2":
		r = UsWest2
	case "us-west-3":
		r = UsWest3
	default:
		err = fmt.Errorf("failed to parse region from '%s'", s)
	}
	return
}

func (v *Region) UnmarshalJSON(buf []byte) (err error) {
	*v, err = ParseRegion(string(bytes.Trim(buf, "\"")))
	return
}

func (v Region) MarshalJSON() (buf []byte, err error) {
	s := fmt.Sprintf("\"%s\"", v.String())
	return []byte(s), nil
}

func (v Region) String() string {
	switch v {
	case AsiaNortheast1:
		return "asia-northeast-1"
	case AsiaNortheast2:
		return "asia-northeast-2"
	case AsiaSouth1:
		return "asia-south-1"
	case AustraliaEast1:
		return "australia-east-1"
	case EuropeCentral1:
		return "europe-central-1"
	case MeWest1:
		return "me-west-1"
	case TestEast1:
		return "test-east-1"
	case TestWest1:
		return "test-west-1"
	case UsEast1:
		return "us-east-1"
	case UsEast2:
		return "us-east-2"
	case UsEast3:
		return "us-east-3"
	case UsMidwest1:
		return "us-midwest-1"
	case UsMidwest2:
		return "us-midwest-2"
	case UsSouth1:
		return "us-south-1"
	case UsSouth2:
		return "us-south-2"
	case UsSouth3:
		return "us-south-3"
	case UsWest1:
		return "us-west-1"
	case UsWest2:
		return "us-west-2"
	case UsWest3:
		return "us-west-3"
	default:
		return ""
	}
}

// SecurityGroupRuleProtocol defines model for SecurityGroupRuleProtocol.
type SecurityGroupRuleProtocol int

// Defines values for SecurityGroupRuleProtocol.
const (
	All SecurityGroupRuleProtocol = iota
	Icmp
	Tcp
	Udp
)

func (v *SecurityGroupRuleProtocol) UnmarshalJSON(buf []byte) error {
	switch strings.Trim(string(buf), "\"") {
	case "all":
		*v = All
	case "icmp":
		*v = Icmp
	case "tcp":
		*v = Tcp
	case "udp":
		*v = Udp
	default:
		return fmt.Errorf("failed to unmarshal json for SecurityGroupRuleProtocol: %s", string(buf))
	}
	return nil
}
func (v *SecurityGroupRuleProtocol) String() string {
	switch *v {

	case All:
		return "all"
	case Icmp:
		return "icmp"
	case Tcp:
		return "tcp"
	case Udp:
		return "udp"
	default:
		return ""
	}
}

// UserStatus Status of the user's account.
type UserStatus int

// Defines values for UserStatus.
const (
	UserStatusActive UserStatus = iota
	UserStatusDeactivated
)

func (v *UserStatus) UnmarshalJSON(buf []byte) error {
	switch strings.Trim(string(buf), "\"") {
	case "active":
		*v = UserStatusActive
	case "deactivated":
		*v = UserStatusDeactivated
	default:
		return fmt.Errorf("failed to unmarshal json for UserStatus: %s", string(buf))
	}
	return nil
}
func (v *UserStatus) String() string {
	switch *v {

	case UserStatusActive:
		return "active"
	case UserStatusDeactivated:
		return "deactivated"
	default:
		return ""
	}
}

// AddSSHKeyRequest defines model for AddSSHKeyRequest.
type AddSSHKeyRequest struct {
	// Name The name of the SSH key.
	Name string `json:"name"`

	// PublicKey The public key for the SSH key.
	PublicKey *string `json:"public_key,omitempty"`
}

// ApiErrorAccountInactive defines model for ApiErrorAccountInactive.
type Error struct {
	// Code The unique identifier for the type of error.
	Code string `json:"code"`

	// Message A description of the error.
	Message string `json:"message"`

	// Suggestion One or more suggestions of possible ways to fix the error.
	Suggestion string `json:"suggestion"`
}

// Filesystem Information about a shared filesystem.
type Filesystem struct {
	// BytesUsed The approximate amount of storage used by the filesystem in bytes. This estimate is
	// updated every few hours.
	BytesUsed *int `json:"bytes_used,omitempty"`

	// Created The date and time at which the filesystem was created. Formatted as an ISO 8601 timestamp.
	Created time.Time `json:"created"`

	// CreatedBy Information about a user in your Team.
	CreatedBy User `json:"created_by"`

	// ID The unique identifier (ID) of the filesystem.
	ID string `json:"id"`

	// IsInUse Whether the filesystem is currently in use by an instance. Filesystems that
	// are in use cannot be deleted.
	IsInUse bool `json:"is_in_use"`

	// MountPoint The absolute path indicating where on instances the filesystem will be mounted.
	MountPoint string `json:"mount_point"`

	// Name The name of the filesystem.
	Name   string            `json:"name"`
	Region RegionDescription `json:"region"`
}

// FilesystemCreateRequest defines model for FilesystemCreateRequest.
type FilesystemCreateRequest struct {
	// Name The name of the filesystem.
	Name   string `json:"name"`
	Region Region `json:"region"`
}

// FilesystemDeleteResponse defines model for FilesystemDeleteResponse.
type FilesystemDeleteResponse struct {
	// DeletedIds The unique identifiers of the filesystems that were deleted.
	DeletedIds []string `json:"deleted_ids"`
}

// FirewallRule defines model for FirewallRule.
type FirewallRule struct {
	// Description A human-readable description of the rule.
	Description string `json:"description"`

	// PortRange An inclusive range of network ports specified as `[min, max]`.
	// Not allowed for the `icmp` protocol but required for the others.
	//
	// To specify a single port, list it twice (for example, `[22,22]`).
	PortRange *[]int                    `json:"port_range,omitempty"`
	Protocol  SecurityGroupRuleProtocol `json:"protocol"`

	// SourceNetwork The set of source IPv4 addresses from which you want to allow inbound
	// traffic. These addresses must be specified in CIDR notation. You can
	// specify individual public IPv4 CIDR blocks such as `1.2.3.4` or
	// `1.2.3.4/32`, or you can specify `0.0.0.0/0` to allow access from any
	// address.
	//
	// This value is a string consisting of a public IPv4 address optionally
	// followed by a slash (/) and an integer mask (the network prefix).
	// If no mask is provided, the API assumes `/32` by default.
	SourceNetwork string `json:"source_network"`
}

// GeneratedSSHKey Information about a server-generated SSH key, which can be used to access instances over
// SSH.
type GeneratedSSHKey struct {
	// ID The unique identifier (ID) of the SSH key.
	ID string `json:"id"`

	// Name The name of the SSH key.
	Name string `json:"name"`

	// PrivateKey The private key generated in the SSH key pair. Make sure to store a
	// copy of this key locally, as Lambda does not store the key server-side.
	PrivateKey string `json:"private_key"`

	// PublicKey The public key for the SSH key.
	PublicKey string `json:"public_key"`
}

// Image An available machine image in Lambda Cloud.
type Image struct {
	Architecture ImageArchitecture `json:"architecture"`

	// CreatedTime The date and time that the image was created.
	CreatedTime time.Time `json:"created_time"`

	// Description Additional information about the image.
	Description string `json:"description"`

	// Family The family the image belongs to.
	Family string `json:"family"`

	// ID The unique identifier (ID) for an image.
	ID string `json:"id"`

	// Name The human-readable identifier for an image.
	Name   string            `json:"name"`
	Region RegionDescription `json:"region"`

	// UpdatedTime The date and time that the image was last updated.
	UpdatedTime time.Time `json:"updated_time"`

	// Version The image version.
	Version string `json:"version"`
}

// ImageSpecificationFamily Specifies the image to use by its family name.
type ImageSpecificationFamily struct {
	// Family The family name of the image.
	Family string `json:"family"`
}

// ImageSpecificationID Specifies the image to use by its unique identifier.
type ImageSpecificationID struct {
	ID string `json:"id"`
}

// Instance Detailed information about the instance.
type Instance struct {
	Actions InstanceActionAvailability `json:"actions"`

	// FileSystemNames The names of the filesystems attached to the instance. If no filesystems are attached, this array is empty.
	FileSystemNames []string `json:"file_system_names"`

	// Hostname The hostname assigned to this instance, which resolves to the instance's IP.
	Hostname *string `json:"hostname,omitempty"`

	// ID The unique identifier of the instance.
	ID            string        `json:"id"`
	InstanceQuote InstanceQuote `json:"instance_type"`

	// IP The public IPv4 address of the instance.
	IP *string `json:"ip,omitempty"`

	// JupyterToken The secret token used to log into the JupyterLab server hosted on the instance.
	JupyterToken *string `json:"jupyter_token,omitempty"`

	// JupyterURL The URL that opens the JupyterLab environment on the instance.
	JupyterURL *string `json:"jupyter_url,omitempty"`

	// Name If set, the user-provided name of the instance.
	Name *string `json:"name,omitempty"`

	// PrivateIP The private IPv4 address of the instance.
	PrivateIP *string           `json:"private_ip,omitempty"`
	Region    RegionDescription `json:"region"`

	// SSHKeyNames The names of the SSH keys that are allowed to access the instance.
	SSHKeyNames []string `json:"ssh_key_names"`

	// Status The current status of the instance.
	Status Status `json:"status"`
}

// InstanceLaunchRequest defines model for InstanceLaunchRequest.
type InstanceLaunchRequest struct {
	// FileSystemNames The names of the filesystems you want to attach to the instance.
	// Currently, you can attach only one filesystem during instance creation.
	// By default, no filesystems are attached.
	FileSystems []string `json:"file_system_names,omitempty"`

	// Image The machine image you want to use. Defaults to the latest Lambda Stack image.
	Image *Image `json:"image,omitempty"`

	// InstanceTypeName The type of instance you want to launch. To retrieve a list of available instance types, see
	// [List available instance types](#get-/api/v1/instance-types).
	Model string `json:"instance_type_name"`

	// Name The name you want to assign to your instance. Must be 64 characters or fewer.
	Name   string `json:"name,omitempty"`
	Region Region `json:"region_name"`

	// SSHKeyNames The names of the SSH keys you want to use to provide access to the instance.
	// Currently, exactly one SSH key must be specified.
	SSHKeyNames []string `json:"ssh_key_names"`

	// Data An instance configuration string specified in a valid
	// [cloud-init user-data](https://cloudinit.readthedocs.io/en/latest/explanation/format.html)
	// format. You can use this field to configure your instance on launch. The
	// user data string must be plain text and cannot exceed 1MB in size.
	Data string `json:"user_data,omitempty"`
}

// InstanceActionAvailability defines model for InstanceActionAvailability.
type InstanceActionAvailability struct {
	ColdReboot InstanceActionAvailabilityDetails `json:"cold_reboot"`
	Migrate    InstanceActionAvailabilityDetails `json:"migrate"`
	Rebuild    InstanceActionAvailabilityDetails `json:"rebuild"`
	Restart    InstanceActionAvailabilityDetails `json:"restart"`
	Terminate  InstanceActionAvailabilityDetails `json:"terminate"`
}

// InstanceActionAvailabilityDetails defines model for InstanceActionAvailabilityDetails.
type InstanceActionAvailabilityDetails struct {
	// Available If set, indicates that the relevant operation can be performed on the instance in its current state.
	Available bool `json:"available"`

	// ReasonCode A code representing the instance state that is blocking the operation. Only provided if the operation is blocked.
	ReasonCode *string `json:"reason_code,omitempty"`

	// ReasonDescription A longer description of why this operation is currently blocked. Only provided if the operation is blocked.
	ReasonDescription *string `json:"reason_description,omitempty"`
}

type InstanceQuote struct {
	// Description A description of the instance type.
	Description string `json:"description"`

	// GpuDescription The type of GPU used by this instance type.
	GpuDescription string `json:"gpu_description"`

	// Name The name of the instance type.
	Name string `json:"name"`

	// PriceCentsPerHour The price of the instance type in US cents per hour.
	PriceCentsPerHour int               `json:"price_cents_per_hour"`
	Specs             InstanceTypeSpecs `json:"specs"`
}

// InstanceTypeSpecs defines model for InstanceTypeSpecs.
type InstanceTypeSpecs struct {
	// Gpus The number of GPUs.
	Gpus int `json:"gpus"`

	// MemoryGib The amount of RAM in gibibytes (GiB).
	MemoryGib int `json:"memory_gib"`

	// StorageGib The amount of storage in gibibytes (GiB).
	StorageGib int `json:"storage_gib"`

	// Vcpus The number of virtual CPUs.
	Vcpus int `json:"vcpus"`
}

type InstanceTypes map[string]InstanceTypesItem

type InstanceTypesItem struct {
	InstanceQuote InstanceQuote `json:"instance_type"`

	// Regions A list of the regions in which this instance type is available.
	Regions []RegionDescription `json:"regions_with_capacity_available"`
}

// Region defines model for Region.
type RegionDescription struct {
	// Description The location represented by the region code.
	Description string `json:"description"`
	Name        Region `json:"name"`
}

// SSHKey Information about a stored SSH key, which can be used to access instances over
// SSH.
type SSHKey struct {
	// ID The unique identifier (ID) of the SSH key.
	ID string `json:"id"`

	// Name The name of the SSH key.
	Name string `json:"name"`

	// PublicKey The public key for the SSH key.
	PublicKey string `json:"public_key"`
}

// User Information about a user in your Team.
type User struct {
	// Email The email address of the user.
	Email string `json:"email"`

	// ID The unique identifier for the user.
	ID string `json:"id"`

	// Status Status of the user's account.
	Status UserStatus `json:"status"`
}

/*
func (img Image) MarshalJSON() ([]byte, error) {
	return json.Marshal(moderations.InputImageURL{
		Type: "image_url",
		ImageURL: moderations.ImageURL{
			URL: string(img),
		},
	})
}
*/

type Client struct {
	key    string
	client *http.Client
}

func NewClient(client *http.Client, key string) (*Client, error) {
	c := &Client{
		key:    key,
		client: client,
	}
	return c, nil
}

func (c *Client) NewJSONRequest(ctx context.Context, method string, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, "https://cloud.lambdalabs.com/api/v1/"+path, body)
	if err != nil {
		return nil, err
	}

	h := req.Header
	h.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.key))+":")
	h.Add("Content-Type", "application/json")
	h.Add("Accept", "application/json")

	return req, nil
}

func (c *Client) Instances() ([]Instance, error) {
	req, err := c.NewJSONRequest(context.Background(), "GET", "instances", nil)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if int(res.StatusCode) < 200 || int(res.StatusCode) >= 300 {
		return nil, fmt.Errorf("response not ok %d, %+v", res.StatusCode, res)
	}

	dec := json.NewDecoder(res.Body)

	var response struct {
		Data []Instance `json:"data"`
	}

	if err = dec.Decode(&response); err != nil {
		return nil, err
	}

	res.Body.Close()

	return response.Data, nil
}

type Title struct {
	region Region
	model  string
}

func NewTitle(region Region, model string) Title {
	return Title{
		region,
		model,
	}
}

func ParseTitle(s string) (t Title, err error) {
	i := strings.Index(s, "/")
	if i == -1 {
		err = fmt.Errorf("'/' not found")
		return
	}
	t.region, err = ParseRegion(s[:i])
	if err != nil {
		return
	}
	t.model = s[i+1:]
	return
}

func (t Title) String() string {
	return fmt.Sprintf("%s/%s", t.region.String(), t.model)
}

func (t1 Title) Less(t2 Title) bool {
	if t1.region != t2.region {
		return t1.region.String() < t2.region.String()
	}
	return t1.model < t2.model
}

func (t Title) Model() string {
	return t.model
}

func (t Title) Region() Region {
	return t.region
}

func (c *Client) Availability() (quotes map[Title]InstanceQuote, titles []Title, err error) {
	var req *http.Request
	req, err = c.NewJSONRequest(context.Background(), "GET", "instance-types", nil)
	if err != nil {
		return
	}

	var res *http.Response
	res, err = c.client.Do(req)
	if err != nil {
		return
	}

	if int(res.StatusCode) < 200 || int(res.StatusCode) >= 300 {
		err = fmt.Errorf("response not ok %d, %+v", res.StatusCode, res)
		return
	}

	dec := json.NewDecoder(res.Body)

	var response struct {
		Data InstanceTypes `json:"data"`
	}

	if err = dec.Decode(&response); err != nil {
		return
	}

	quotes = make(map[Title]InstanceQuote)
	for _, e := range response.Data {
		quote := e.InstanceQuote
		for _, r := range e.Regions {
			title := NewTitle(r.Name, quote.Name)
			titles = append(titles, title)
			quotes[title] = quote
		}
	}

	res.Body.Close()

	return
}

func ParseKey(key []byte) (string, error) {
	if !bytes.HasPrefix(key, []byte("ssh-")) {
		return "", fmt.Errorf("missing SSH prefix")
	}

	unprintable := func(r rune) bool {
		return !(unicode.IsDigit(r) ||
			unicode.IsLetter(r) ||
			unicode.IsSymbol(r) ||
			unicode.IsMark(r) ||
			unicode.IsPunct(r) ||
			unicode.IsSpace(r))
	}

	if bytes.ContainsFunc(key, unprintable) {
		return "", fmt.Errorf("contains non-printable characters")
	}

	f := bytes.Fields(key)

	if len(f) < 2 {
		return "", fmt.Errorf("too short")
	}

	/*
		if len(f) > 4 {
			return "", fmt.Errorf("too many fields")
		}
	*/
	if len(f[1]) > 1000 {
		return "", fmt.Errorf("hash string too long")
	}
	return string(append(append(f[0], byte(' ')), f[1]...)), nil
}

type Strings []string

func (s Strings) Error() string {
	return strings.Join(s, "\n")
}

func (c *Client) SSHKeys() (keys map[string][]string, fetch, parse error) {
	req, err := c.NewJSONRequest(context.Background(), "GET", "ssh-keys", nil)
	if err != nil {
		fetch = err
		return
	}

	res, err := c.client.Do(req)
	if err != nil {
		fetch = err
		return
	}

	if int(res.StatusCode) < 200 || int(res.StatusCode) >= 300 {
		fetch = fmt.Errorf("response not ok %d, %+v", res.StatusCode, res)
		return
	}

	dec := json.NewDecoder(res.Body)

	var response struct {
		Data []SSHKey `json:"data"`
	}

	if err = dec.Decode(&response); err != nil {
		fetch = err
		return
	}

	keys = make(map[string][]string)
	var errs []string
	for _, key := range response.Data {
		k, err := ParseKey([]byte(key.PublicKey))
		if err != nil {
			errs = append(errs, fmt.Sprintf("error parsing cloud SSH key '%s': %+v", key.Name, err))
			continue
		}
		keys[k] = append(keys[k], key.Name)
	}

	if len(errs) != 0 {
		parse = Strings(errs)
	}

	return
}

func (c *Client) Launch(title Title, name string, keys, filesystems []string, data string) (map[string]struct{}, error) {
	r, w := io.Pipe()
	req, err := c.NewJSONRequest(context.Background(), "POST", "instance-operations/launch", r)
	if err != nil {
		return nil, err
	}

	body := InstanceLaunchRequest{
		SSHKeyNames: keys,
		Model:       title.Model(),
		Region:      title.Region(),
		FileSystems: filesystems,
	}

	go func() {
		defer w.Close()
		enc := json.NewEncoder(w)
		err = enc.Encode(body)
	}()

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if int(res.StatusCode) < 200 || int(res.StatusCode) >= 300 {
		return nil, fmt.Errorf("response not ok %d, %+v", res.StatusCode, res)
	}

	dec := json.NewDecoder(res.Body)

	var response struct {
		Data struct {
			IDs []string `json:"instance_ids"`
		} `json:"data"`
	}

	if err = dec.Decode(&response); err != nil {
		return nil, err
	}

	ids := make(map[string]struct{})
	for _, id := range response.Data.IDs {
		ids[id] = struct{}{}
	}

	return ids, nil
}

func (c *Client) Terminate(ids []string) (error) {
	r, w := io.Pipe()
	req, err := c.NewJSONRequest(context.Background(), "POST", "instance-operations/terminate", r)
	if err != nil {
		return err
	}

	type Body struct {
		InstanceIDs []string `json:"instance_ids"`
	}

	body := Body {
		ids,
	}

	go func() {
		defer w.Close()
		enc := json.NewEncoder(w)
		err = enc.Encode(body)
	}()

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if int(res.StatusCode) < 200 || int(res.StatusCode) >= 300 {
		return fmt.Errorf("response not ok %d, %+v", res.StatusCode, res)
	}

	return nil
}
