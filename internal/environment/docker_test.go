package environment

import (
	"reflect"
	"testing"
)

func TestDockerComposeProjectNameSanitizesExerciseName(t *testing.T) {
	got := DockerComposeProjectName(" Jerry Root Container! ")
	want := "gymctl-jerry-root-container"
	if got != want {
		t.Fatalf("DockerComposeProjectName() = %q, want %q", got, want)
	}
}

func TestDockerComposeTeardownProjectNamesIncludesLegacyName(t *testing.T) {
	got := dockerComposeTeardownProjectNames("jerry-root-container")
	want := []string{"gymctl-jerry-root-container", DockerLegacyComposeProjectName}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dockerComposeTeardownProjectNames() = %#v, want %#v", got, want)
	}
}
