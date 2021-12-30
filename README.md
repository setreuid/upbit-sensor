# Upbit auto trading bot



## How to
1. Golang ([download](https://go.dev/doc/install))
2. Input 'Access key', 'Secret key' ([get](https://upbit.com/mypage/open_api_management))
3. Build `go build`



## Set Upbit's API key
1. Open web site "https://upbit.com/mypage/open_api_management"
2. Check all permissions
3. Input your ip address
![key](https://user-images.githubusercontent.com/11921622/147729103-0371824a-aac4-4b02-b718-b9db72c59c8e.png)
4. Issuance of authentication key
5. Input keys (see https://github.com/setreuid/upbit-sensor/blob/main/tools.go#L133)
![key2](https://user-images.githubusercontent.com/11921622/147729911-4fc96ea0-fae0-4ebd-a3f0-d249c26b87fc.png)



## Decide which stocks to trade
See https://github.com/setreuid/upbit-sensor/blob/main/main.go#L19

![stocks](https://user-images.githubusercontent.com/11921622/147730035-2613a8f6-8004-41f4-84cd-3bc39328fba6.png)


## Modify trading options
See https://github.com/setreuid/upbit-sensor/blob/main/main.go#L35

![options](https://user-images.githubusercontent.com/11921622/147730114-1b1d25f6-37d6-44ae-8452-dd64ae62bba2.png)


## Modify trading strategy
See https://github.com/setreuid/upbit-sensor/blob/main/upbit.go#L230

![trade](https://user-images.githubusercontent.com/11921622/147730038-9a487169-519f-48b2-bddb-be67da903490.png)
