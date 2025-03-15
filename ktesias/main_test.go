package ktesias

import (
	"context"
	"embed"
	"fmt"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/google/uuid"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/aristides/diplomat"
	pb "github.com/odysseia-greek/delphi/aristides/proto"
	"os"
	"strings"
	"testing"
)

type contextKey string

const createdResourcesKey contextKey = "createdResources"

type Resource struct {
	Kind string
	Name string
}

func addResourceToContext(ctx context.Context, resource Resource) context.Context {
	existing, ok := ctx.Value(createdResourcesKey).([]Resource)
	if !ok {
		existing = []Resource{}
	}
	existing = append(existing, resource)
	return context.WithValue(ctx, createdResourcesKey, existing)
}

func getResourcesFromContext(ctx context.Context) []Resource {
	resources, ok := ctx.Value(createdResourcesKey).([]Resource)
	if !ok {
		return []Resource{}
	}
	return resources
}

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "progress", // can define default values
}

//go:embed features/*.feature
var featureFiles embed.FS

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
}

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {

		//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=HIPPOKRATES
		logging.System(`
 __  _ ______    ___  _____ ____   ____  _____
|  |/ ]      |  /  _]/ ___/|    | /    |/ ___/
|  ' /|      | /  [_(   \_  |  | |  o  (   \_ 
|    \|_|  |_||    _]\__  | |  | |     |\__  |
|     | |  |  |   [_ /  \ | |  | |  _  |/  \ |
|  .  | |  |  |     |\    | |  | |  |  |\    |
|__|\_| |__|  |_____| \___||____||__|__| \___|
`)
		logging.System("\"Κτησίας δὲ, ἰατρὸς μὲν τῷ ἔργῳ, ἀλλ' ἐν τῷ γράφειν φανταστικώτερος καὶ παραδοξολογίας μεστός.\"")
		logging.System("\"Ktesias, though a physician in profession, is more fanciful in writing and full of marvels and exaggeration.\"")
		logging.System("starting test suite setup.....")

		logging.System("getting env variables and creating config")
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	odysseia, err := New()
	if err != nil {
		logging.Error(fmt.Sprintf("error creating odysseia instance: %v", err))
		os.Exit(1)
	}

	// perikles
	ctx.Step(`^a secret should be created for tls certs for host "([^"]*)"$`, odysseia.aSecretShouldBeCreatedForTlsCertsForHost)
	ctx.Step(`^CiliumNetWorkPolicies should exist for role "([^"]*)" from host "([^"]*)"$`, odysseia.ciliumNetWorkPoliciesShouldExistForRoleFromHost)
	ctx.Step(`^the created resource "([^"]*)" is checked after a wait$`, odysseia.theCreatedResourceIsCheckedAfterAWait)
	ctx.Step(`^a CiliumNetWorkPolicy should exist for access from the deployment "([^"]*)" to the host "([^"]*)"$`, odysseia.aCiliumNetWorkPolicyShouldExistForAccessFromTheDeploymentToTheHost)
	ctx.Step(`^a deployment is created with role "([^"]*)", access "([^"]*)", host "([^"]*)" and being a client of "([^"]*)"$`, odysseia.aDeploymentIsCreatedWithRoleAccessHostAndBeingAClientOf)

	// solon
	ctx.Step(`^solon returns a healthy response$`, odysseia.solonReturnsAHealthyResponse)
	ctx.Step(`^a one time token is returned$`, odysseia.aOneTimeTokenIsReturned)
	ctx.Step(`^a request is made to register the running pod with incorrect role and access annotations$`, odysseia.aRequestIsMadeToRegisterTheRunningPodWithIncorrectRoleAndAccessAnnotations)
	ctx.Step(`^a request is made to register the running pod with correct role and access annotations but a mismatched podname$`, odysseia.aRequestIsMadeToRegisterTheRunningPodWithCorrectRoleAndAccessAnnotationsButAMismatchedPodname)
	ctx.Step(`^a validation error is returned that the pod annotations do not match the requested role and access$`, odysseia.aValidationErrorIsReturnedThatThePodAnnotationsDoNotMatchTheRequestedRoleAndAccess)
	ctx.Step(`^a request is made to register the running pod with correct role and access annotations$`, odysseia.aRequestIsMadeToRegisterTheRunningPodWithCorrectRoleAndAccessAnnotations)
	ctx.Step(`^a successful register should be made$`, odysseia.aSuccessfulRegisterShouldBeMade)
	ctx.Step(`^a request is made for a second one time token$`, odysseia.aRequestIsMadeForASecondOneTimeToken)
	ctx.Step(`^a request is made for a one time token$`, odysseia.aRequestIsMadeForAOneTimeToken)
	ctx.Step(`^the token from the actual podname is valid$`, odysseia.theTokenFromTheActualPodnameIsValid)
	ctx.Step(`^the token from the mismatched podname is not valid$`, odysseia.theTokenFromTheMismatchedPodnameIsNotValid)
	ctx.Step(`^the tokens are not usable twice$`, odysseia.theTokensAreNotUsableTwice)

	// flow
	ctx.Step(`^aristides is asked for the current config$`, odysseia.aristidesIsAskedForTheCurrentConfig)
	ctx.Step(`^a call is made to the correct index with the correct action$`, odysseia.aCallIsMadeToTheCorrectIndexWithTheCorrectAction)
	ctx.Step(`^an elastic client is created with the one time token that was created$`, odysseia.anElasticClientIsCreatedWithTheOneTimeTokenThatWasCreated)
	ctx.Step(`^a (\d+) should be returned$`, odysseia.aShouldBeReturned)
	ctx.Step(`^an elastic client is created with the vault data$`, odysseia.anElasticClientIsCreatedWithTheVaultData)
	ctx.Step(`^a call is made to an index not part of the annotations$`, odysseia.aCallIsMadeToAnIndexNotPartOfTheAnnotations)

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		odysseia.cleanupResources()

		return ctx, nil
	})
}

func TestMain(m *testing.M) {
	format := "pretty"
	var tag string // Initialize an empty slice to store the tags

	for _, arg := range os.Args[1:] {
		if arg == "-test.v=true" {
			format = "progress"
		} else if strings.HasPrefix(arg, "-tags=") {
			tagsString := strings.TrimPrefix(arg, "-tags=")
			tag = strings.Split(tagsString, ",")[0]
		}
	}

	opts := godog.Options{
		Format:          format,
		FeatureContents: getFeatureContents(), // Get the embedded feature files
	}

	if tag != "" {
		opts.Tags = tag
	}

	status := godog.TestSuite{
		Name:                 "godogs",
		TestSuiteInitializer: InitializeTestSuite,
		ScenarioInitializer:  InitializeScenario,
		Options:              &opts,
	}.Run()

	ambassador, err := diplomat.NewClientAmbassador(diplomat.DEFAULTADDRESS)
	if err != nil {
		logging.Error(fmt.Sprintf("Unable to create ambassador client: %v", err))
	}

	healthy := ambassador.WaitForHealthyState()
	if !healthy {
		logging.Info("aristides service not ready - restarting seems the only option")
		os.Exit(1)
	}

	uuidCode := uuid.New().String()
	_, err = ambassador.ShutDown(context.Background(), &pb.ShutDownRequest{Code: uuidCode})
	if err != nil {
		logging.Error(err.Error())
	}

	os.Exit(status)
}

func getFeatureContents() []godog.Feature {
	features := []godog.Feature{}
	featureFileNames, _ := featureFiles.ReadDir("features")
	for _, file := range featureFileNames {
		if !file.IsDir() && file.Name() != "README.md" { // Skip directories and README.md if any
			filePath := fmt.Sprintf("features/%s", file.Name())
			fileContent, _ := featureFiles.ReadFile(filePath)
			features = append(features, godog.Feature{Name: file.Name(), Contents: fileContent})
		}
	}
	return features
}
