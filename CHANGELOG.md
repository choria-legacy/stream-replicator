|Date      |Issue |Description                                                                                              |
|----------|------|---------------------------------------------------------------------------------------------------------|
|2018/01/29|65    |Correctly list advisory target modes in the puppet module as `source` and `target`                       |
|2018/01/29|      |Release 0.0.6                                                                                            |
|2018/01/29|58    |Ability to send advisories about tracked nodes when they stop sending metadata                           |
|2018/01/17|51    |Treat sysconfig file as configuration on el6                                                             |
|2018/01/17|      |Release 0.0.5                                                                                            |
|2018/01/16|      |Rework packaging to use new choria packaging framework                                                   |
|2018/01/03|27    |During condrestart on el6 handle all known pids not just configured ones                                 |
|2018/01/04|      |Release 0.0.4                                                                                            |
|2018/01/03|38    |Improve uniqueness of client names to make multi site deployments easier                                 |
|2018/01/03|35    |Complete support for managing el6 systems with Puppet                                                    |
|2018/01/03|34    |Use os.family so that the configuration applies to all EL like systems                                   |
|2018/01/03|33    |When making a release set the Puppet module ensure value to match the version being released             |
|2018/01/03|      |Release 0.0.3                                                                                            |
|2018/01/03|27    |Stop all running services rather than configured ones on el6                                             |
|2018/01/01|10    |Adds a Puppet module                                                                                     |
|2018/01/01|24    |Write the last-seen cache tempfile in the statedir to support multi partition machines                   |
|2018/01/01|      |Release 0.0.2                                                                                            |
|2018/01/01|16    |Save the last-seen data regularly and on shutdown.  Restore state on startup                             |
|2017/12/31|19    |Record via prometheus the size of messages received and published                                        |
|2017/12/31|20    |Correctly track last seen date to ensure more than 1 update is sent                                      |
|2017/12/29|5     |Initial, incomplete, SSL support                                                                         |
|2017/12/27|11    |Make various sleeps interruptable to speed up shutdown                                                   |
|2017/12/27|      |Release 0.0.1                                                                                            |
