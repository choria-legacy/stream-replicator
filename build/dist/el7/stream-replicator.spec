%define debug_package %{nil}
%define pkgname {{pkgname}}
%define version {{version}}
%define bindir {{bindir}}
%define etcdir {{etcdir}}
%define iteration {{iteration}}
%define dist {{dist}}
%define manage_conf {{manage_conf}}
%define go_arch {{go_arch}}
%define target_arch {{arch}}

Name: %{pkgname}
Version: %{version}
Release: %{iteration}.%{dist}
Summary: The Choria NATS Streaming Topic Replicator
License: Apache-2.0
URL: https://choria.io
Group: System Tools
Packager: R.I.Pienaar <rip@devco.net>
Source0: %{pkgname}-%{version}.tgz
BuildRoot: %{_tmppath}/%{pkgname}-%{version}-%{release}-root-%(%{__id_u} -n)

%description
Replicator for NATS Streaming Topics

%prep
%setup -q

%build

%install
rm -rf %{buildroot}
%{__install} -d -m0755  %{buildroot}/usr/lib/systemd/system
%{__install} -d -m0755  %{buildroot}/etc/logrotate.d
%{__install} -d -m0755  %{buildroot}%{bindir}
%{__install} -d -m0755  %{buildroot}%{etcdir}
%{__install} -d -m0755  %{buildroot}/var/log
%{__install} -d -m0756  %{buildroot}/var/lib/%{pkgname}
%{__install} -m0644 dist/stream-replicator@.service %{buildroot}/usr/lib/systemd/system/%{pkgname}@.service
%{__install} -m0644 dist/stream-replicator-logrotate %{buildroot}/etc/logrotate.d/%{pkgname}
%if 0%{?manage_conf} > 0
%{__install} -m0640 dist/sr.yaml %{buildroot}%{etcdir}/sr.yaml
%endif
%{__install} -m0755 stream-replicator-%{version}-linux-%{go_arch} %{buildroot}%{bindir}/%{pkgname}
touch %{buildroot}/var/log/%{pkgname}.log

%clean
rm -rf %{buildroot}

%post
if [ $1 -eq 1 ] ; then
  systemctl --no-reload preset %{pkgname} >/dev/null 2>&1 || :
fi

/bin/systemctl --system daemon-reload >/dev/null 2>&1 || :

if [ $1 -ge 1 ]; then
  /bin/systemctl try-restart %{pkgname} >/dev/null 2>&1 || :;
fi

%preun
if [ $1 -eq 0 ] ; then
  systemctl --no-reload disable --now %{pkgname} >/dev/null 2>&1 || :
fi

%files
%if 0%{?manage_conf} > 0
%attr(640, root, nobody) %config(noreplace)%{etcdir}/sr.yaml
%endif
%{bindir}/%{pkgname}
/etc/logrotate.d/%{pkgname}
/usr/lib/systemd/system/%{pkgname}@.service
%attr(640, nobody, nobody)/var/log/%{pkgname}.log
%attr(640, nobody, nobody)/var/lib/%{pkgname}

%changelog
* Tue Dec 26 2017 R.I.Pienaar <rip@devco.net>
- Initial Release
