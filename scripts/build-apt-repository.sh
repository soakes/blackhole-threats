#!/usr/bin/env bash
set -euo pipefail

site_root="${1:?site output directory is required}"
version="${2:?version is required}"
commit="${3:?commit is required}"
build_date="${4:?build date is required}"
suite="${5:-stable}"
component="${6:-main}"
package="$(dpkg-parsechangelog -SSource)"
package_version="$(dpkg-parsechangelog -SVersion)"
upstream_version="${package_version%-*}"
package_group="${package:0:1}"
pool_dir="${site_root}/pool/${component}/${package_group}/${package}"
dist_root="dists/${suite}"
release_architectures="amd64 arm64 armhf"
architectures=(amd64 arm64 armhf)

mkdir -p "${pool_dir}"

if [ ! -f "../${package}_${upstream_version}.orig.tar.gz" ]; then
  echo "Creating upstream source tarball ${package}_${upstream_version}.orig.tar.gz"
  git archive \
    --format=tar.gz \
    --prefix="${package}-${upstream_version}/" \
    -o "../${package}_${upstream_version}.orig.tar.gz" \
    HEAD
fi

echo "Building Debian source package"
dpkg-buildpackage -us -uc -S -sa

find .. -maxdepth 1 -type f \
  \( \
    -name '*.dsc' -o \
    -name '*.orig.tar.*' -o \
    -name '*.debian.tar.*' -o \
    -name '*.diff.gz' \
  \) \
  -exec mv {} "${pool_dir}/" \;
find .. -maxdepth 1 -type f \( -name '*.changes' -o -name '*.buildinfo' \) -delete

for arch in "${architectures[@]}"; do
  echo "Building Debian package for ${arch}"
  # These package builds cross-compile a static Go binary, so native build deps are sufficient.
  DEB_BUILD_OPTIONS=nocheck \
  DEB_HOST_ARCH="${arch}" \
  BLACKHOLE_LDFLAGS="-X github.com/soakes/blackhole-threats/internal/buildinfo.Version=${version} -X github.com/soakes/blackhole-threats/internal/buildinfo.Commit=${commit} -X github.com/soakes/blackhole-threats/internal/buildinfo.BuildDate=${build_date}" \
    dpkg-buildpackage -d -us -uc -b -a"${arch}"

  find .. -maxdepth 1 -type f -name '*.deb' -exec mv {} "${pool_dir}/" \;
  find .. -maxdepth 1 -type f \( -name '*.changes' -o -name '*.buildinfo' \) -delete
done

pushd "${site_root}" >/dev/null
for arch in "${architectures[@]}"; do
  binary_dir="dists/${suite}/${component}/binary-${arch}"
  mkdir -p "${binary_dir}"
  dpkg-scanpackages -a "${arch}" pool /dev/null > "${binary_dir}/Packages"
  gzip -9c "${binary_dir}/Packages" > "${binary_dir}/Packages.gz"
  xz -9c "${binary_dir}/Packages" > "${binary_dir}/Packages.xz"
done

source_dir="dists/${suite}/${component}/source"
mkdir -p "${source_dir}"
dpkg-scansources pool /dev/null > "${source_dir}/Sources"
gzip -9c "${source_dir}/Sources" > "${source_dir}/Sources.gz"
xz -9c "${source_dir}/Sources" > "${source_dir}/Sources.xz"

apt-ftparchive \
  -o "APT::FTPArchive::Release::Origin=${package}" \
  -o "APT::FTPArchive::Release::Label=${package}" \
  -o "APT::FTPArchive::Release::Suite=${suite}" \
  -o "APT::FTPArchive::Release::Codename=${suite}" \
  -o "APT::FTPArchive::Release::Components=${component}" \
  -o "APT::FTPArchive::Release::Architectures=${release_architectures}" \
  -o "APT::FTPArchive::Release::Acquire-By-Hash=yes" \
  release "${dist_root}" > "${dist_root}/Release"
popd >/dev/null
