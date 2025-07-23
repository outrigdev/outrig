{
  stdenv,
  lib,
  fetchurl,
}:
let
  system = lib.splitString "-" stdenv.hostPlatform.system;
  arch = lib.head system;
  platform =
    let
      platform = lib.head (lib.tail system);
    in
    if platform == "linux" then
      "Linux"
    else if platform == "darwin" then
      "Darwin"
    else
      platform;

  # Platform-specific hashes for v0.9.0-alpha.0
  hashes = {
    "x86_64-linux" = "a44e6c21ad290ae8beab6b8cb3541e400355b86358c8b68075c9c4280601e0f6";
    "aarch64-linux" = "8e08c33f1e31b393d102795f18299654cbebdcfd39ffb3e655c10a5882594405";
    "x86_64-darwin" = "a33c96c6dfd69604278e460895bb8f69b94f5fcc289a79386d46cbb44290cffc";
    "aarch64-darwin" = "25c0891228357920244eb09c19d7bbff6fe6f1624d9237659d6c1e96b99fc1ad";
  };
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.9.0-alpha.0";

  src = fetchurl {
    url = "https://github.com/outrigdev/outrig/releases/download/v${version}/${pname}_${version}_${platform}_${arch}.tar.gz";
    sha256 = hashes.${stdenv.hostPlatform.system};
  };

  sourceRoot = "${pname}_${version}_${platform}_${arch}";

  unpackPhase = ''
    tar -xzf $src
  '';

  dontBuild = true;

  installPhase = ''
    mkdir -p $out/bin
    cp ${pname} $out/bin/${pname}
    chmod +x $out/bin/${pname}
  '';

  meta = {
    description = "Dev-time observability tool for Go programs. Search logs, monitor goroutines, and track variables";
    homepage = "https://outrig.run/";
    license = lib.licenses.asl20;
    maintainers = with lib.maintainers; [ sawka ];
    mainProgram = "outrig";
  };
})
