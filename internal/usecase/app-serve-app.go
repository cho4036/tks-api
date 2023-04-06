package usecase

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/openinfradev/tks-api/internal/repository"
	argowf "github.com/openinfradev/tks-api/pkg/argo-client"
	"github.com/openinfradev/tks-api/pkg/domain"
	"github.com/openinfradev/tks-api/pkg/log"
)

type IAppServeAppUsecase interface {
	CreateAppServeApp(app *domain.AppServeApp) (appId string, taskId string, err error)
	GetAppServeApps(organizationId string, showAll bool) ([]domain.AppServeApp, error)
	GetAppServeAppById(appId string) (*domain.AppServeApp, error)
	DeleteAppServeApp(appId string) (res string, err error)
	UpdateAppServeApp(app *domain.AppServeAppTask) (ret string, err error)
	PromoteAppServeApp(appId string) (ret string, err error)
	AbortAppServeApp(appId string) (ret string, err error)
}

type AppServeAppUsecase struct {
	repo repository.IAppServeAppRepository
	argo argowf.ArgoClient
}

func NewAppServeAppUsecase(r repository.IAppServeAppRepository, argoClient argowf.ArgoClient) IAppServeAppUsecase {
	return &AppServeAppUsecase{
		repo: r,
		argo: argoClient,
	}
}

func (u *AppServeAppUsecase) CreateAppServeApp(app *domain.AppServeApp) (string, string, error) {
	if app == nil {
		return "", "", fmt.Errorf("invalid app obj")
	}

	// For type 'build' and 'all', imageUrl and executablePath
	// are constructed based on pre-defined rule
	// (Refer to 'tks-appserve-template')
	if app.Type != "deploy" {
		// Validate param
		if app.AppServeAppTasks[0].ArtifactUrl == "" {
			return "", "", fmt.Errorf("error: For 'build'/'all' type task, 'artifact_url' is mandatory param")
		}

		// Construct imageUrl
		imageUrl := viper.GetString("image-registry-url") + "/" + app.Name + ":" + app.AppServeAppTasks[0].Version
		app.AppServeAppTasks[0].ImageUrl = imageUrl

		if app.AppType == "springboot" {
			// Construct executable_path
			artiUrl := app.AppServeAppTasks[0].ArtifactUrl
			tempArr := strings.Split(artiUrl, "/")
			exeFilename := tempArr[len(tempArr)-1]

			executablePath := "/usr/src/myapp/" + exeFilename
			app.AppServeAppTasks[0].ExecutablePath = executablePath
		}
	} else {
		// Validate param for 'deploy' type.
		// TODO: check params for legacy spring app case
		if app.AppType == "springboot" {
			if app.AppServeAppTasks[0].ImageUrl == "" || app.AppServeAppTasks[0].ExecutablePath == "" ||
				app.AppServeAppTasks[0].Profile == "" || app.AppServeAppTasks[0].ResourceSpec == "" {
				return "",
					"",
					fmt.Errorf("Error: For 'deploy' type task, the following params must be provided." +
						"\n\t- image_url\n\t- executable_path\n\t- profile\n\t- resource_spec")
			}
		}
	}

	// TODO: Validate PV params

	appId, taskId, err := u.repo.CreateAppServeApp(app)
	if err != nil {
		return "", "", err
	}

	fmt.Printf("appId = %s, taskId = %s", appId, taskId)

	// Call argo workflow
	workflow := "serve-java-app"

	opts := argowf.SubmitOptions{}
	opts.Parameters = []string{
		"type=" + app.Type,
		"strategy=" + app.AppServeAppTasks[0].Strategy,
		"app_type=" + app.AppType,
		"target_cluster_id=" + app.TargetClusterId,
		"app_name=" + app.Name,
		"asa_id=" + appId,
		"asa_task_id=" + taskId,
		"artifact_url=" + app.AppServeAppTasks[0].ArtifactUrl,
		"image_url=" + app.AppServeAppTasks[0].ImageUrl,
		"port=" + app.AppServeAppTasks[0].Port,
		"profile=" + app.AppServeAppTasks[0].Profile,
		"extra_env=" + app.AppServeAppTasks[0].ExtraEnv,
		"app_config=" + app.AppServeAppTasks[0].AppConfig,
		"app_secret=" + app.AppServeAppTasks[0].AppSecret,
		"resource_spec=" + app.AppServeAppTasks[0].ResourceSpec,
		"executable_path=" + app.AppServeAppTasks[0].ExecutablePath,
		"git_repo_url=" + viper.GetString("git-repository-url"),
		"harbor_pw_secret=" + viper.GetString("harbor-pw-secret"),
		"pv_enabled=" + strconv.FormatBool(app.AppServeAppTasks[0].PvEnabled),
		"pv_storage_class=" + app.AppServeAppTasks[0].PvStorageClass,
		"pv_access_mode=" + app.AppServeAppTasks[0].PvAccessMode,
		"pv_size=" + app.AppServeAppTasks[0].PvSize,
		"pv_mount_path=" + app.AppServeAppTasks[0].PvMountPath,
	}

	log.Info("Submitting workflow: ", workflow)

	workflowId, err := u.argo.SumbitWorkflowFromWftpl(workflow, opts)
	if err != nil {
		return "", "", fmt.Errorf("failed to submit workflow. Err: %s", err)
	}
	log.Info("Successfully submitted workflow: ", workflowId)

	return appId, app.Name, nil
}

func (u *AppServeAppUsecase) GetAppServeApps(organizationId string, showAll bool) ([]domain.AppServeApp, error) {
	apps, err := u.repo.GetAppServeApps(organizationId, showAll)
	if err != nil {
		fmt.Println(apps)
	}

	return apps, nil
}

func (u *AppServeAppUsecase) GetAppServeAppById(appId string) (*domain.AppServeApp, error) {
	app, err := u.repo.GetAppServeAppById(appId)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (u *AppServeAppUsecase) DeleteAppServeApp(appId string) (res string, err error) {
	app, err := u.repo.GetAppServeAppById(appId)
	if err != nil {
		return "", fmt.Errorf("error while getting ASA Info from DB. Err: %s", err)
	}

	// Validate app status
	if app.Status == "WAIT_FOR_PROMOTE" || app.Status == "BLUEGREEN_FAILED" {
		return "", fmt.Errorf("the app is in blue-green related state. Promote or abort first before deleting")
	}

	/********************
	 * Start delete task *
	 ********************/

	appTask := &domain.AppServeAppTask{
		AppServeAppId: app.ID,
		Version:       "",
		ArtifactUrl:   "",
		ImageUrl:      "",
		Status:        "DELETING",
		Profile:       "",
		Output:        "",
		CreatedAt:     time.Now(),
	}

	taskId, err := u.repo.CreateTask(appTask)
	if err != nil {
		log.Info("taskId = [%s]", taskId)
		log.Error("Failed to create delete task. Err:", err)
		return "", fmt.Errorf("failed to create delete task. Err: %s", err)
	}

	workflow := "delete-java-app"
	log.Info("Submitting workflow: ", workflow)

	workflowId, err := u.argo.SumbitWorkflowFromWftpl(workflow, argowf.SubmitOptions{
		Parameters: []string{
			"type=" + app.Type,
			"target_cluster_id=" + app.TargetClusterId,
			"app_name=" + app.Name,
			"asa_id=" + app.ID,
			"asa_task_id=" + taskId,
			"artifact_url=" + "NA",
			"image_url=" + "NA",
			"port=" + "NA",
			"profile=" + "NA",
			"resource_spec=" + "NA",
			"executable_path=" + "NA",
		},
	})
	if err != nil {
		log.Error("Failed to submit workflow. Err:", err)
		return "", fmt.Errorf("failed to submit workflow. Err: %s", err)
	}
	log.Info("Successfully submitted workflow: ", workflowId)

	return fmt.Sprintf("The app %s is being deleted. "+
		"Confirm result by checking the app status after a while.", app.Name), nil
}

func (u *AppServeAppUsecase) UpdateAppServeApp(appTask *domain.AppServeAppTask) (ret string, err error) {
	if appTask == nil {
		return "", errors.New("invalid parameters. appTask is nil")
	}

	log.Info("Starting normal update process..")

	// TODO: for more strict validation, check if immutable fields are provided by user
	// and those values are changed or not. (name, type, app_type, target_cluster)

	// Validate 'strategy' param
	if !(appTask.Strategy == "rolling-update" || appTask.Strategy == "blue-green" || appTask.Strategy == "canary") {
		return "", fmt.Errorf("Error: 'strategy' should be one of these values." +
			"\n\t- rolling-update\n\t- blue-green\n\t- canary")
	}

	app, err := u.repo.GetAppServeAppById(appTask.AppServeAppId)
	if err != nil {
		return "", fmt.Errorf("error while getting ASA Info from DB. Err: %s", err)
	}

	if app.Type != "deploy" {
		// Construct imageUrl
		imageUrl := viper.GetString("image-registry-url") + "/" + app.Name + ":" + appTask.Version
		appTask.ImageUrl = imageUrl

		// Construct executable_path
		if app.AppType == "springboot" {
			artiUrl := appTask.ArtifactUrl
			tempArr := strings.Split(artiUrl, "/")
			exeFilename := tempArr[len(tempArr)-1]

			executablePath := "/usr/src/myapp/" + exeFilename
			appTask.ExecutablePath = executablePath
		}
	}

	// 'Update' GRPC only creates ASA Task record
	taskId, err := u.repo.CreateTask(appTask)
	if err != nil {
		log.Info("taskId = [%s]", taskId)
		return "", fmt.Errorf("failed to update app-serve application. Err: %s", err)
	}

	// Call argo workflow
	workflow := "serve-java-app"

	log.Info("Submitting workflow: ", workflow)

	workflowId, err := u.argo.SumbitWorkflowFromWftpl(workflow, argowf.SubmitOptions{
		Parameters: []string{
			"type=" + app.Type,
			"strategy=" + appTask.Strategy,
			"app_type=" + app.AppType,
			"target_cluster_id=" + app.TargetClusterId,
			"app_name=" + app.Name,
			"asa_id=" + app.ID,
			"asa_task_id=" + taskId,
			"artifact_url=" + appTask.ArtifactUrl,
			"image_url=" + appTask.ImageUrl,
			"port=" + appTask.Port,
			"profile=" + appTask.Profile,
			"extra_env=" + appTask.ExtraEnv,
			"app_config=" + appTask.AppConfig,
			"app_secret=" + appTask.AppSecret,
			"resource_spec=" + appTask.ResourceSpec,
			"executable_path=" + appTask.ExecutablePath,
			"git_repo_url=" + viper.GetString("git-repository-url"),
			"harbor_pw_secret=" + viper.GetString("harbor-pw-secret"),
			"pv_enabled=" + strconv.FormatBool(appTask.PvEnabled),
			"pv_storage_class=" + appTask.PvStorageClass,
			"pv_access_mode=" + appTask.PvAccessMode,
			"pv_size=" + appTask.PvSize,
			"pv_mount_path=" + appTask.PvMountPath,
		},
	})
	if err != nil {
		log.Error("Failed to submit workflow. Err:", err)
		return "", fmt.Errorf("failed to submit workflow. Err: %s", err)
	}
	log.Info("Successfully submitted workflow: ", workflowId)

	return fmt.Sprintf("The app '%s' is being updated. "+
		"Confirm result by checking the app status after a while.", app.Name), nil
}

func (u *AppServeAppUsecase) PromoteAppServeApp(appId string) (ret string, err error) {
	app, err := u.repo.GetAppServeAppById(appId)
	if err != nil {
		return "", fmt.Errorf("error while getting ASA Info from DB. Err: %s", err)
	}

	if app.Status != "WAIT_FOR_PROMOTE" {
		return "", fmt.Errorf("the app is not in 'WAIT_FOR_PROMOTE' state. Exiting")
	}

	// Get the latest task ID so that the task status can be modified inside workflow once the promotion is done.
	latestTaskId := app.AppServeAppTasks[0].ID
	log.Info("latestTaskId = ", latestTaskId)

	// Call argo workflow
	workflow := "promote-java-app"

	log.Info("Submitting workflow: ", workflow)

	workflowId, err := u.argo.SumbitWorkflowFromWftpl(workflow, argowf.SubmitOptions{
		Parameters: []string{
			"target_cluster_id=" + app.TargetClusterId,
			"app_name=" + app.Name,
			"asa_id=" + app.ID,
			"asa_task_id=" + latestTaskId,
		},
	})
	if err != nil {
		log.Error("failed to submit workflow. Err:", err)
		return "", fmt.Errorf("failed to submit workflow. Err: %s", err)
	}
	log.Info("Successfully submitted workflow: ", workflowId)

	return fmt.Sprintf("The app '%s' is being promoted. "+
		"Confirm result by checking the app status after a while.", app.Name), nil

}

func (u *AppServeAppUsecase) AbortAppServeApp(appId string) (ret string, err error) {
	app, err := u.repo.GetAppServeAppById(appId)
	if err != nil {
		return "", fmt.Errorf("error while getting ASA Info from DB. Err: %s", err)
	}

	if app.Status != "WAIT_FOR_PROMOTE" && app.Status != "BLUEGREEN_FAILED" {
		return "", fmt.Errorf("the app is not in blue-green related state. Exiting")
	}

	// Get the latest task ID so that the task status can be modified inside workflow once the promotion is done.
	latestTaskId := app.AppServeAppTasks[0].ID
	log.Info("latestTaskId = ", latestTaskId)

	// Call argo workflow
	workflow := "abort-java-app"

	log.Info("Submitting workflow: ", workflow)

	// Call argo workflow
	workflowId, err := u.argo.SumbitWorkflowFromWftpl(workflow, argowf.SubmitOptions{
		Parameters: []string{
			"target_cluster_id=" + app.TargetClusterId,
			"app_name=" + app.Name,
			"asa_id=" + app.ID,
			"asa_task_id=" + latestTaskId,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to submit workflow. Err: %s", err)
	}
	log.Info("Successfully submitted workflow: ", workflowId)

	return fmt.Sprintf("The app '%s' is being promoted. "+
		"Confirm result by checking the app status after a while.", app.Name), nil
}
