%define debug_package %{nil}
%define pkgname {{pkgname}}
%define version {{version}}
%define bindir {{bindir}}
%define etcdir {{etcdir}}
%define iteration {{iteration}}
%define dist {{dist}}
%define manage_conf {{manage_conf}}

Name: %{pkgname}
Version: %{version}
Release: %{iteration}.%{dist}
Summary: The Choria NATS Streaming Topic Replicator
License: Apache-2.0
URL: https://choria.io
Group: System Tools
Packager: R.I.Pienaar <rip@devco.net>
Source0: %{pkgname}-%{version}-Linux-amd64.tgz
BuildRoot: %{_tmppath}/%{pkgname}-%{version}-%{release}-root-%(%{__id_u} -n)

%description
Replicator for NATS Streaming Topics

%prep
%setup -q

%build

%install
rm -rf %{buildroot}
%{__install} -d -m0755  %{buildroot}/etc/sysconfig
%{__install} -d -m0755  %{buildroot}/etc/init.d
%{__install} -d -m0755  %{buildroot}/etc/logrotate.d
%{__install} -d -m0755  %{buildroot}%{bindir}
%{__install} -d -m0755  %{buildroot}%{etcdir}
%{__install} -d -m0755  %{buildroot}/var/log
%{__install} -d -m0756  %{buildroot}/var/run/%{pkgname}
%{__install} -d -m0756  %{buildroot}/var/lib/%{pkgname}
%{__install} -m0644 dist/stream-replicator.init %{buildroot}/etc/init.d/%{pkgname}
%{__install} -m0644 dist/stream-replicator.sysconfig %{buildroot}/etc/sysconfig/%{pkgname}
%{__install} -m0644 dist/stream-replicator-logrotate %{buildroot}/etc/logrotate.d/%{pkgname}
%if 0%{?manage_conf} > 0
%{__install} -m0640 dist/sr.yaml %{buildroot}%{etcdir}/sr.yaml
%endif
%{__install} -m0755 stream-replicator-%{version}-Linux-amd64 %{buildroot}%{bindir}/%{pkgname}
touch %{buildroot}/var/log/%{pkgname}.log

%clean
rm -rf %{buildroot}

%post
/sbin/chkconfig --add %{pkgname} || :

%postun
if [ "$1" -ge 1 ]; then
  /sbin/service %{pkgname} condrestart &>/dev/null || :
fi

%preun
if [ "$1" = 0 ] ; then
  /sbin/service %{pkgname} stop > /dev/null 2>&1
  /sbin/chkconfig --del %{pkgname} || :
fi

%files
%if 0%{?manage_conf} > 0
%attr(640, root, nobody) %config(noreplace)%{etcdir}/sr.yaml
%endif
%{bindir}/%{pkgname}
/etc/logrotate.d/%{pkgname}
%attr(755, root, root)/etc/init.d/%{pkgname}
%attr(644, root, root)/etc/sysconfig/%{pkgname}
%attr(755, nobody, nobody)/var/run/%{pkgname}
%attr(640, nobody, nobody)/var/log/%{pkgname}.log
%attr(640, nobody, nobody)/var/lib/%{pkgname}

%changelog
* Tue Dec 26 2017 R.I.Pienaar <rip@devco.net>
- Initial Release
