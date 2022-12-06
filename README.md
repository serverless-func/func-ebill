## Function e-Bill

> 信用卡交易记录邮件解析

## 招商银行

- 每日信用管家（新）
- 信用管家消费提醒（旧）

## Development

```shell
fission spec init
fission fn create --spec --name func-ebill --src src.zip --entrypoint Handler --env go --buildcmd "./customBuild.sh"
fission route create --spec --method GET --method POST --name func-ebill --url /{Subpath} --function func-ebill --createingress  --ingressrule "ebill.func.dongfg.com=/*" --ingresstls "tls-ebill-func-dongfg" --ingressannotation "cert-manager.io/cluster-issuer=letsencrypt-dongfg" --ingressannotation "nginx.ingress.kubernetes.io/use-regex=true"
```