def dockerRegName
def dockerImageName
def dockerImageVersion

pipeline {
    agent any
    stages {
        stage('检出') {
            steps {
                checkout([$class: 'GitSCM',
                    branches: [[name: GIT_BUILD_REF]],
                    userRemoteConfigs: [[
                        url: GIT_REPO_URL,
                        credentialsId: CREDENTIALS_ID
                    ]]])
            }
        }

        stage('加载缓存') {
            when {
                expression {
                    fileExists(DOCKER_CACHE_PATH).equals(true)
                }
            }
            steps {
                sh "docker load -i ${DOCKER_CACHE_PATH}"
            }
        }

        stage('镜像构建') {
            steps {
                script {
                    dockerRegName = System.getenv('DOCKER_REG_NAME') ?: 'docker'
                    dockerImageName = "${PROJECT_NAME.toLowerCase()}/${dockerRegName}/${DEPOT_NAME.toLowerCase()}"
                    dockerImageVersion = env.GIT_TAG ?: 'latest'
                    docker.withRegistry("${DOCKER_REG_HOST}", "${env.CODING_ARTIFACTS_CREDENTIALS_ID}") {
                        docker.build("${dockerImageName}:${dockerImageVersion}").push()
                        docker.build("${dockerImageName}:latest").push()
                    }
                }
            }
        }

        stage('镜像部署') {
            steps {
                withKubeConfig([credentialsId: "${KUBE_CONFIG_ID}"]) {
                    useCustomStepPlugin(key: 'kube_deploy', version: 'latest', params: [app:"${DEPOT_NAME.toLowerCase()}", version:"${GIT_TAG}"])
                }
            }
        }

        stage('更新缓存') {
            when {
                expression {
                    fileExists(DOCKER_CACHE_PATH).equals(false)
                }
            }
            steps {
                sh 'mkdir -p /root/.cache/docker/'
                sh "docker save -o ${DOCKER_CACHE_PATH} ${dockerImageName}:latest"
            }
        }
    }
    environment {
        DOCKER_REG_HOST = "https://${CCI_CURRENT_TEAM}-docker.pkg.${CCI_CURRENT_DOMAIN}"
        DOCKER_CACHE_PATH = "/root/.cache/docker/${DEPOT_NAME.toLowerCase()}.tar"
    }
}
