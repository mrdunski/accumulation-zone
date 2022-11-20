pipeline {
    environment {
        IMAGE_TAG = "${(GIT_BRANCH == 'main') ? "0.$BUILD_NUMBER.0.$GIT_COMMIT" : "$GIT_COMMIT"}"
        IMAGE_REPO_AND_TAG = "git.kende.pl/kende/accumulation-zone:$IMAGE_TAG"
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
            }
        }

        stage('Build docker') {
            steps {
                script {
                    currentBuild.description = "image: $IMAGE_REPO_AND_TAG"
                }
                withCredentials([usernamePassword(credentialsId: 'gitea', passwordVariable: 'chartPassword', usernameVariable: 'chartUser')]) {
                    sh "docker login --username $chartUser --password $chartPassword git.kende.pl"
                }
                sh "docker build --pull -t $IMAGE_REPO_AND_TAG ."
                sh "docker push $IMAGE_REPO_AND_TAG"
            }
        }

    }
}