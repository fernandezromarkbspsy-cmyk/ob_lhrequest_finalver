(function () {
  var isLocalFrontend = window.location.hostname === "localhost" || window.location.hostname === "127.0.0.1";
  var isSeparateLocalPort = isLocalFrontend && window.location.port && window.location.port !== "8080";

  window.SOC5_CONFIG = {
    API_BASE: isSeparateLocalPort ? "http://localhost:8080" : ""
  };
})();
