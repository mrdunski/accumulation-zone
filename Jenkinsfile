pipeline {
    environment {
        IMAGE_TAG = "${(GIT_BRANCH == 'main') ? "0.$BUILD_NUMBER.0.$GIT_COMMIT" : "$GIT_COMMIT"}"
        IMAGE_REPO = "mrdunski/accumulation-zone"
        IMAGE_REPO_AND_TAG = "$IMAGE_REPO:$IMAGE_TAG"
        CHART_VERSION = "${(GIT_BRANCH == 'main') ? "1.$BUILD_NUMBER.0+$GIT_COMMIT" : "0.0.1+$GIT_COMMIT"}"
        DOCKER_BUILDKIT = "1"
        CHART_REPOSITORY_URL="https://git.kende.pl/api/packages/kende/helm/api/charts"
    }
    agent {
        node { label 'docker' }
    }

    stages {
        stage('Test') {
            steps {
                sh "docker build  --pull --no-cache -o build --target artifacts ."
                junit 'build/report.xml'
                cobertura autoUpdateHealth: false, autoUpdateStability: false, coberturaReportFile: 'build/coverprofile.xml', conditionalCoverageTargets: '70, 0, 0', failUnhealthy: false, failUnstable: false, lineCoverageTargets: '80, 0, 0', maxNumberOfBuilds: 0, methodCoverageTargets: '80, 0, 0', onlyStable: false, sourceEncoding: 'ASCII', zoomCoverageChart: false
            }
        }

        stage('Build docker') {
            steps {
                script {
                    currentBuild.description = "image: $IMAGE_REPO_AND_TAG"
                }
                withCredentials([usernamePassword(credentialsId: 'dockerhub', passwordVariable: 'pwd', usernameVariable: 'usr')]) {
                    sh "docker login --username $usr --password $pwd"
                }
                sh "docker build --pull -t $IMAGE_REPO_AND_TAG ."
                sh "docker push $IMAGE_REPO_AND_TAG"
            }
        }

        stage('Tag Latest') {
            when {
                branch 'main'
            }
            steps {
                script {
                    currentBuild.description = "image: $IMAGE_REPO_AND_TAG"
                }
                withCredentials([usernamePassword(credentialsId: 'dockerhub', passwordVariable: 'pwd', usernameVariable: 'usr')]) {
                    sh "docker login --username $usr --password $pwd"
                }
                sh "docker tag $IMAGE_REPO_AND_TAG $IMAGE_REPO"
                sh "docker push $IMAGE_REPO"
            }
        }

        stage('Create chart') {
            steps {
                script {
                    currentBuild.description = """image: $IMAGE_REPO_AND_TAG
chart-version: $CHART_VERSION
app-version: $IMAGE_TAG"""
                }
                sh "apk add --no-cache curl tar gzip"
                sh "curl https://get.helm.sh/helm-v3.1.0-linux-amd64.tar.gz --output helm.tar.gz; tar -zxvf helm.tar.gz; cp linux-amd64/helm /usr/local/bin; rm helm.tar.gz linux-amd64 -r"
                sh "helm package ./chart/accumulation-zone --version $CHART_VERSION --app-version $IMAGE_TAG"
                withCredentials([usernamePassword(credentialsId: 'gitea', passwordVariable: 'chartPassword', usernameVariable: 'chartUser')]) {
                    sh "curl --user $chartUser:$chartPassword -X POST --upload-file ./accumulation-zone-${CHART_VERSION}.tgz ${CHART_REPOSITORY_URL}"
                }
            }
        }
    }
}