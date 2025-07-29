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

  # Platform-specific hashes for v0.9.0
  hashes = {
    "x86_64-linux" = "d2eeaa8cf3d8b64b57b5638f2df8219c38c8f10063765f8e9b045bc22c355781";
    "aarch64-linux" = "7e5f7f28bbbcad069673aae6f93a14083f61969533441f8feb23c12ede90fff8";
    "x86_64-darwin" = "97d474f7f4e371b6986f69c6fa09a43a75e4b9f22939a93512b9ce8a38d3d780";
    "aarch64-darwin" = "053818a443b91cc93840f28f0d7d92603e0f73f184d7fd15499c7536759a98fd";
  };
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.9.0";

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
