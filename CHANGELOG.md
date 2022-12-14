# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.3] - 2021-11-15

### Bugfix

- PDF 解析库更换，修复账单文件升级的解析问题
- 账单文件升级导致的字段对应问题修复

## [1.2.2] - 2021-09-02

### Bugfix

- 修复退款记录解析问题

## [1.2.1] - 2021-07-02

### Added

- 使用 Coding 持续集成服务部署 

## [1.2.0] - 2021-07-02

### Bugfix

- 修复 2021-05-26 之后的邮件解析问题

## [1.1.1] - 2020-11-18

### Bugfix

- 上传文件临时写入/tmp, 云函数仅该目录有访问权限
- 修复异常文件未删除的问题

## [1.1.0] - 2020-11-18

### Added

- 解析招行pdf月账单

## [1.0.2] - 2020-11-12

### Bugfix

- 修复超过三位数金额解析问题

## [1.0.1] - 2020-09-22

### Bugfix

- Lower case field in result json

## [1.0.0] - 2020-09-22

### Added

- Parse cmb email
- Automatically build and deploy with github workflow depend on [funcraft](https://github.com/alibaba/funcraft)
- Automatically publish github release with [goreleaser](https://goreleaser.com/)