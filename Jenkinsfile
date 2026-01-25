pipeline {
    agent any

    options {
        disableConcurrentBuilds()
    }

    stages {
        stage('Checkout'){
            steps {
                checkout scm
            }
        }
        stage('Prep buildx') {
            steps {
                script {
                    env.BUILDX_BUILDER = getBuildxBuilder();
                }
            }
        }
        stage('Build full Image') {
            steps {
                withCredentials([usernamePassword(credentialsId: 'dockerhub', usernameVariable: 'DOCKERHUB_CREDENTIALS_USR', passwordVariable: 'DOCKERHUB_CREDENTIALS_PSW')]) {
                    sh 'docker login -u $DOCKERHUB_CREDENTIALS_USR -p "$DOCKERHUB_CREDENTIALS_PSW"'
                }
                sh """
                    docker buildx build \
                        --pull \
                        --builder \$BUILDX_BUILDER  \
                        --platform linux/arm64 \
                        --target full \
                        -t nbr23/jacadi:latest \
                        -t nbr23/jacadi:full \
                        -t nbr23/jacadi:full-`git rev-parse --short HEAD` \
                        ${ "$GIT_BRANCH" == "master" ? "--push" : ""} .
                    """
            }
        }
        stage('Build light Image') {
            steps {
                withCredentials([usernamePassword(credentialsId: 'dockerhub', usernameVariable: 'DOCKERHUB_CREDENTIALS_USR', passwordVariable: 'DOCKERHUB_CREDENTIALS_PSW')]) {
                    sh 'docker login -u $DOCKERHUB_CREDENTIALS_USR -p "$DOCKERHUB_CREDENTIALS_PSW"'
                }
                sh """
                    docker buildx build \
                        --pull \
                        --builder \$BUILDX_BUILDER  \
                        --platform linux/arm64 \
                        --target full \
                        -t nbr23/jacadi:light \
                        -t nbr23/jacadi:light-`git rev-parse --short HEAD` \
                        ${ "$GIT_BRANCH" == "master" ? "--push" : ""} .
                    """
            }
        }
        stage('Sync github repo') {
            when { branch 'master' }
            steps {
                syncRemoteBranch('git@github.com:nbr23/jacadi.git', 'master')
            }
        }
    }
    post {
        always {
            sh 'docker buildx stop $BUILDX_BUILDER || true'
            sh 'docker buildx rm $BUILDX_BUILDER || true'
        }
    }
}
