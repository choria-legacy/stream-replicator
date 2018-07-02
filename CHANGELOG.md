|Date      |Issue |Description                                                                                              |
|----------|------|---------------------------------------------------------------------------------------------------------|
|2018/07/02|      |Release 0.3.0                                                                                            |
|2018/07/02|85    |Redo the connection error handling attempt to be more robust, , requires NATS Streaming Server >= 0.10.0 |
|2018/06/25|      |Release 0.2.0                                                                                            |
|2018/06/25|85    |Improve the robustness of connection error handling, requires NATS Streaming Server >= 0.10.0            |
|2018/05/28|88    |Support Choria SSL - including SSL enrollment, Puppet CA and manual CA support                           |
|2018/05/18|86    |Allow data producers to include hints that data should be replicated immediately                         |
|2018/04/11|      |Release 0.1.1                                                                                            |
|2018/04/11|82    |Do not zero the logfile on startup                                                                       |
|2018/04/11|80    |Fix startup announce message ordering                                                                    |
|2018/04/11|      |Release 0.1.0                                                                                            |
|2018/04/11|76    |Do not rotate empty log files                                                                            |
|2018/04/11|75    |Announce version, config and topic at startup                                                            |
|2018/04/04|      |Release 0.0.8                                                                                            |
|2018/04/04|71    |Ensure the service restarts correctly when managed by Puppet                                             |
|2018/01/29|      |Release 0.0.7                                                                                            |
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
