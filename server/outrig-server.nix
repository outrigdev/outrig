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

  # Platform-specific hashes for v0.9.1-beta.0
  hashes = {
    "x86_64-linux" = "1e893577e27d755089e9b443b620e381e3974958fffcb3f4255feb530b21bd29";
    "aarch64-linux" = "769b0e9bc658fb3eac43600a56e3334b232bdd98a84315b803dac720dcb32475";
    "x86_64-darwin" = "838bf1c12995424f96751bbd94d0bea8a8b4b02b7453064deedaa329371b88d1";
    "aarch64-darwin" = "b724f2d280fff81d83b4b8025bb91c1ffdbb3566d5ccad7d87e8396ef19b51a4";
  };
in
(stdenv.mkDerivation rec {
  pname = "outrig";
  version = "0.9.1-beta.0";

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
