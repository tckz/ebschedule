ebschedule
===

Update/diff schedule of Amazon EventBridge Scheduler.


# Usage

## update

Update or create schedule.

```
Usage:
  ebschedule update [flags]

Flags:
      --create-schedule-group   create schedule group if not exist (default true)
  -h, --help                    help for update
      --schedule string         path/to/schedule.yaml
```

 - When schedule group does not exist, try to create it if `--create-schedule-group` is `true`.


## diff

Diff schedule between local and remote.

```
Usage:
  ebschedule diff [flags]

Flags:
  -h, --help              help for diff
      --schedule string   path/to/schedule.yaml
```

```
--- arn:aws:scheduler:ap-northeast-1:99999:schedule/default/hello-task
+++ ./schedule.yml
@@ -6,7 +6,7 @@
 GroupName: default
 KmsKeyArn: null
 Name: hello-task
-ScheduleExpression: cron(*/3 * * * ? *)
+ScheduleExpression: cron(*/5 * * * ? *)
 ScheduleExpressionTimezone: Asia/Tokyo
 StartDate: null
 State: DISABLED
```

# schedule.yaml

 - You can generate template of `schedule.yaml` by AWS CLI v2
   ```
   aws scheduler create-schedule --generate-cli-skeleton yaml-input
   ```
 - `ebschedule` read `schedule.yaml` using [go-config](https://github.com/kayac/go-config),
So, You can embed value of environment variables in `schedule.yaml`.
    ```yaml
    Some:
      Key: '{{ env `ENV_VAR_NAME` `default_value` }}'
      PanicIfUndefined: '{{ must_env `ENV_VAR_NAME` }}'
    ```

# Author 

Copyright (c) 2023 tckz <at.tckz@gmail.com>

# LICENSE

MIT License
