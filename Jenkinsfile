pipeline {
    environment {
        IMAGE_TAG = "${(GIT_BRANCH == 'main') ? "0.$BUILD_NUMBER.0.$GIT_COMMIT" : "$GIT_COMMIT"}"
        IMAGE_REPO_AND_TAG = "mrdunski/accumulation-zone:$IMAGE_TAG"
        CHART_VERSION = "${(GIT_BRANCH == 'main') ? "0.$BUILD_NUMBER.0+$GIT_COMMIT" : "0.0.1+$GIT_COMMIT"}"
        DOCKER_BUILDKIT = "1"
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

    }
}