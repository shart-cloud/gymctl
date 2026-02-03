package scenario

type Exercise struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Metadata   ExerciseMeta `yaml:"metadata"`
	Spec       ExerciseSpec `yaml:"spec"`
}

type ExerciseMeta struct {
	Name  string `yaml:"name"`
	Title string `yaml:"title"`
	Track string `yaml:"track"`
	Week  int    `yaml:"week,omitempty"`
	Order int    `yaml:"order,omitempty"`
}

type ExerciseSpec struct {
	Difficulty       string          `yaml:"difficulty"`
	EstimatedTime    string          `yaml:"estimatedTime,omitempty"`
	Points           int             `yaml:"points,omitempty"`
	Description      string          `yaml:"description"`
	LearningOutcomes []string        `yaml:"learningOutcomes,omitempty"`
	Tags             []string        `yaml:"tags,omitempty"`
	Prerequisites    []string        `yaml:"prerequisites,omitempty"`
	Environment      EnvironmentSpec `yaml:"environment"`
	Checks           []Check         `yaml:"checks"`
	Hints            []Hint          `yaml:"hints"`
	SuccessMessage   string          `yaml:"successMessage,omitempty"`
	NextExercise     string          `yaml:"nextExercise,omitempty"`
	References       []Reference     `yaml:"references,omitempty"`
	Variants         []Variant       `yaml:"variants,omitempty"`
	VariantSelection string          `yaml:"variantSelection,omitempty"`
}

type EnvironmentSpec struct {
	Type        string            `yaml:"type"`
	Kubernetes  *KubernetesSpec   `yaml:"kubernetes,omitempty"`
	Docker      *DockerSpec       `yaml:"docker,omitempty"`
	CustomSetup []CustomSetupStep `yaml:"customSetup,omitempty"`
}

type KubernetesSpec struct {
	CreateCluster  *bool           `yaml:"createCluster,omitempty"`
	KindConfig     string          `yaml:"kindConfig,omitempty"`
	Namespace      string          `yaml:"namespace,omitempty"`
	SetupManifests []string        `yaml:"setupManifests,omitempty"`
	WaitFor        []WaitCondition `yaml:"waitFor,omitempty"`
}

type WaitCondition struct {
	Resource  string `yaml:"resource"`
	Condition string `yaml:"condition"`
	Timeout   string `yaml:"timeout,omitempty"`
}

type DockerSpec struct {
	ComposeFile string            `yaml:"composeFile,omitempty"`
	Containers  []DockerContainer `yaml:"containers,omitempty"`
	CopyFiles   []CopyFile        `yaml:"copyFiles,omitempty"`
}

type DockerContainer struct {
	Name  string   `yaml:"name"`
	Build string   `yaml:"build,omitempty"`
	Image string   `yaml:"image,omitempty"`
	Ports []string `yaml:"ports,omitempty"`
}

type CopyFile struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type Check struct {
	Name           string            `yaml:"name"`
	Type           string            `yaml:"type"`
	Resource       string            `yaml:"resource,omitempty"`
	Namespace      string            `yaml:"namespace,omitempty"`
	Jsonpath       string            `yaml:"jsonpath,omitempty"`
	Operator       string            `yaml:"operator,omitempty"`
	Value          interface{}       `yaml:"value,omitempty"`
	ValueType      string            `yaml:"valueType,omitempty"`
	Condition      string            `yaml:"condition,omitempty"`
	Status         string            `yaml:"status,omitempty"`
	Timeout        string            `yaml:"timeout,omitempty"`
	Script         string            `yaml:"script,omitempty"`
	Selector       string            `yaml:"selector,omitempty"`
	Container      string            `yaml:"container,omitempty"`
	Command        []string          `yaml:"command,omitempty"`
	ExpectExitCode *int              `yaml:"expectExitCode,omitempty"`
	ExpectOutput   *ExpectOutput     `yaml:"expectOutput,omitempty"`
	Image          string            `yaml:"image,omitempty"`
	Property       string            `yaml:"property,omitempty"`
	URL            string            `yaml:"url,omitempty"`
	Method         string            `yaml:"method,omitempty"`
	ExpectStatus   *int              `yaml:"expectStatus,omitempty"`
	ExpectBody     *ExpectBody       `yaml:"expectBody,omitempty"`
	Headers        map[string]string `yaml:"headers,omitempty"`
	Path           string            `yaml:"path,omitempty"`
	Check          string            `yaml:"check,omitempty"`
	Recursive      *bool             `yaml:"recursive,omitempty"`
	Exists         *bool             `yaml:"exists,omitempty"`
}

type ExpectOutput struct {
	Contains    string `yaml:"contains,omitempty"`
	NotContains string `yaml:"notContains,omitempty"`
	Regex       string `yaml:"regex,omitempty"`
}

type ExpectBody struct {
	Contains    string `yaml:"contains,omitempty"`
	NotContains string `yaml:"notContains,omitempty"`
	Regex       string `yaml:"regex,omitempty"`
}

type Hint struct {
	Cost    int    `yaml:"cost"`
	File    string `yaml:"file,omitempty"`
	Content string `yaml:"content,omitempty"`
}

type Reference struct {
	Title string `yaml:"title"`
	URL   string `yaml:"url"`
}

type CustomSetupStep struct {
	Type      string `yaml:"type"`
	Script    string `yaml:"script,omitempty"`
	Condition string `yaml:"condition,omitempty"`
	Timeout   string `yaml:"timeout,omitempty"`
}

type Variant struct {
	Name           string   `yaml:"name"`
	SetupManifests []string `yaml:"setupManifests,omitempty"`
	Checks         []Check  `yaml:"checks,omitempty"`
}
