#include "server/service/fileController.hpp"

#include "server/logging.hpp"

#include <chrono>
#include <cstdlib>
#include <iostream>
#include <fstream>
#include <random>
#include <thread>
#include <sstream>
#include <stdexcept>

namespace fs = std::filesystem;

namespace server::services
{

    namespace
    {
        fs::path getTempDir()
        {
            if (const char *t = std::getenv("TMPDIR"); t && *t)
                return fs::path(t);
            return fs::path("/tmp");
        }

        std::string randomHex(std::size_t len)
        {
            static const char *kHex = "0123456789abcdef";
            thread_local std::mt19937 rng{std::random_device{}()};
            std::uniform_int_distribution<int> dist(0, 15);

            std::string s;
            s.reserve(len);
            for (std::size_t i = 0; i < len; ++i)
                s.push_back(kHex[dist(rng)]);
            return s;
        }

        /** httplib headers.emplace duplicates keys — browsers reject duplicate CORS values. */
        void setHeaderOnce(httplib::Response &res, const std::string &key,
                           const std::string &val)
        {
            auto rng = res.headers.equal_range(key);
            res.headers.erase(rng.first, rng.second);
            res.set_header(key, val);
        }

        constexpr int kMinMaxDownloads = 1;
        constexpr int kMaxMaxDownloads = 100;

        int parseMaxDownloadsField(const httplib::Request &req)
        {
            if (!req.form.has_field("maxDownloads"))
                return 1;

            const std::string raw = req.form.get_field("maxDownloads");
            if (raw.empty())
                return 1;

            try
            {
                const long v = std::stol(raw);
                if (v < kMinMaxDownloads)
                    return kMinMaxDownloads;
                if (v > kMaxMaxDownloads)
                    return kMaxMaxDownloads;
                return static_cast<int>(v);
            }
            catch (...)
            {
                return 1;
            }
        }

    } // namespace

    FileController::FileController(int port)
        : port_(port),
          uploadDir_(getTempDir() / "transfera-uploads")
    {
        std::error_code ec;
        fs::create_directories(uploadDir_, ec);

        // Generous timeouts for large 100 MB+ transfers over slower links
        server_.set_read_timeout(300, 0);
        server_.set_write_timeout(300, 0);

        // Configurable max upload size (default 100 MB + 2 MB multipart header headroom)
        std::size_t maxUploadMb = 100;
        if (const char *env = std::getenv("TRANSFERA_MAX_UPLOAD_MB"); env && *env)
        {
            try
            {
                const unsigned long mb = std::stoul(env);
                if (mb > 0 && mb <= 4096)
                    maxUploadMb = mb;
            }
            catch (...)
            {
            }
        }
        constexpr std::size_t kMultipartHeadroomBytes = 2 * 1024 * 1024;
        server_.set_payload_max_length((maxUploadMb * 1024 * 1024) + kMultipartHeadroomBytes);

        registerRoutes();
        setupHttpLogging();
    }

    FileController::~FileController() { stop(); }

    bool FileController::start()
    {
        if (running_.exchange(true))
            return listen_ok_.load();

        listen_ok_ = false;

        serverThread_ = std::thread([this]
                                    {
                                        if (!server_.bind_to_port("0.0.0.0", port_))
                                        {
                                            server::log::error(
                                                "Could not bind to port " + std::to_string(port_) +
                                                " (is another server already running?)");
                                            server::log::error("Fix: lsof -ti :" +
                                                               std::to_string(port_) +
                                                               " | xargs kill -9");
                                            running_ = false;
                                            return;
                                        }

                                        listen_ok_ = true;
                                        if (!server_.listen_after_bind())
                                        {
                                            server::log::error("API server stopped on port " +
                                                               std::to_string(port_));
                                        }
                                        listen_ok_ = false;
                                        running_ = false; });

        for (int i = 0; i < 50 && !listen_ok_.load(); ++i)
            std::this_thread::sleep_for(std::chrono::milliseconds(20));

        if (!listen_ok_.load())
        {
            running_ = false;
            server_.stop();
            if (serverThread_.joinable())
                serverThread_.join();
            return false;
        }

        server::log::info("Transfera API on http://0.0.0.0:" + std::to_string(port_));
        server::log::info("  GET  http://0.0.0.0:" + std::to_string(port_) + "/api/health");
        server::log::info("  POST http://0.0.0.0:" + std::to_string(port_) + "/api/upload");
        server::log::info("  GET  http://0.0.0.0:" + std::to_string(port_) +
                          "/api/download/<port>");
        server::log::info("upload dir: " + uploadDir_.string());
        return true;
    }

    void FileController::stop()
    {
        const bool wasRunning = running_.exchange(false);
        if (!wasRunning)
        {
            if (serverThread_.joinable())
                serverThread_.join();
            fileSharer_.shutdown();
            return;
        }

        server_.stop();
        if (serverThread_.joinable())
            serverThread_.join();
        fileSharer_.shutdown();
        server::log::info("API server stopped");
    }

    void FileController::registerRoutes()
    {
        server_.Options(R"(/.*)", [this](const httplib::Request &req, httplib::Response &res)
                        { handleCorsOrNotFound(req, res); });

        // Health check for frontend / ops
        server_.Get("/api/health", [this](const httplib::Request &, httplib::Response &res)
                    {
                          applyCorsHeaders(res);
                          res.status = 200;
                          res.set_content(R"({"status":"ok"})", "application/json"); });

        // Routes match Next.js client: /api/upload, /api/download/:port
        server_.Post("/api/upload", [this](const httplib::Request &req, httplib::Response &res)
                     { handleUpload(req, res); });

        server_.Get(R"(/api/download/(\d+))", [this](const httplib::Request &req, httplib::Response &res)
                    { handleDownload(req, res); });

        // Legacy paths (Java-style)
        server_.Post("/upload", [this](const httplib::Request &req, httplib::Response &res)
                     { handleUpload(req, res); });

        server_.Get(R"(/download/(\d+))", [this](const httplib::Request &req, httplib::Response &res)
                    { handleDownload(req, res); });

        server_.set_error_handler([this](const httplib::Request &req, httplib::Response &res)
                                  {
                                      if (res.status == 404)
                                          handleCorsOrNotFound(req, res);
                                      // Route handlers already set CORS; do not call applyCorsHeaders
                                      // again (duplicate Access-Control-Allow-Origin breaks browsers).
                                  });
    }

    void FileController::setupHttpLogging()
    {
        if (!server::log::httpAccessEnabled())
            return;

        server_.set_logger([](const httplib::Request &req, const httplib::Response &res)
                           {
                               std::ostringstream oss;
                               oss << req.method << ' ' << req.path << " -> " << res.status;
                               if (server::log::verboseEnabled())
                               {
                                   if (!req.remote_addr.empty())
                                       oss << " from=" << req.remote_addr;
                                   const auto cl = req.get_header_value("Content-Length");
                                   if (!cl.empty())
                                       oss << " req_bytes=" << cl;
                               }
                               server::log::info(oss.str());
                           });

        server_.set_error_logger([](const httplib::Error err, const httplib::Request *req)
                                   {
                                       std::ostringstream oss;
                                       oss << httplib::to_string(err);
                                       if (req)
                                           oss << ' ' << req->method << ' ' << req->path;
                                       server::log::error(oss.str());
                                   });
    }

    void FileController::applyCorsHeaders(httplib::Response &res) const
    {
        setHeaderOnce(res, "Access-Control-Allow-Origin", "*");
        setHeaderOnce(res, "Access-Control-Allow-Methods", "GET, POST, OPTIONS");
        setHeaderOnce(res, "Access-Control-Allow-Headers",
                      "Content-Type, Authorization, Accept, X-Requested-With, Range");
        setHeaderOnce(res, "Access-Control-Max-Age", "86400");
        setHeaderOnce(res, "Access-Control-Expose-Headers",
                      "Content-Disposition, X-Filename, X-Downloads-Remaining, Content-Type, "
                      "Content-Length, Accept-Ranges, Content-Range");
    }

    void FileController::handleCorsOrNotFound(const httplib::Request &req,
                                              httplib::Response &res) const
    {
        applyCorsHeaders(res);

        if (req.method == "OPTIONS")
        {
            res.status = 204;
            return;
        }

        res.status = 404;
        res.set_content("Not Found", "text/plain");
    }

    std::string FileController::sanitizeDisplayFilename(const std::string &originalFilename)
    {
        fs::path base = fs::path(originalFilename).filename();
        std::string name = base.string();
        if (name.empty() || name == "." || name == "..")
            return "unnamed-file";

        // Hidden files like ".env" — keep leading dot as stem
        if (base.extension().empty() && !name.empty() && name.front() == '.')
            return name;

        return name;
    }

    std::string FileController::makeUniqueName(const std::string &originalFilename)
    {
        const std::string display = sanitizeDisplayFilename(originalFilename);
        const fs::path base = fs::path(display);
        const std::string ext = base.extension().string();
        std::string stem = base.stem().string();
        if (stem.empty())
            stem = "unnamed-file";
        return randomHex(32) + "_" + stem + ext;
    }

    void FileController::handleUpload(const httplib::Request &req, httplib::Response &res)
    {
        applyCorsHeaders(res);

        if (req.method != "POST")
        {
            res.status = 405;
            res.set_content("Method Not Allowed", "text/plain");
            return;
        }

        const std::string contentType = req.get_header_value("Content-Type");
        if (contentType.empty() ||
            contentType.find("multipart/form-data") == std::string::npos)
        {
            res.status = 400;
            res.set_content("Bad Request: Content-Type must be multipart/form-data", "text/plain");
            return;
        }

        try
        {
            if (!req.form.has_file("file"))
            {
                res.status = 400;
                res.set_content(
                    "Bad Request: missing multipart field 'file' (field name must be \"file\")",
                    "text/plain");
                return;
            }

            const httplib::FormData uploaded = req.form.get_file("file");
            if (uploaded.content.empty())
            {
                res.status = 400;
                res.set_content("Bad Request: empty file upload", "text/plain");
                return;
            }

            std::string filename = uploaded.filename;

            const std::string displayName = sanitizeDisplayFilename(filename);

            const std::string uniqueFilename = makeUniqueName(displayName);
            const fs::path filePath = uploadDir_ / uniqueFilename;

            {
                std::ofstream fos(filePath, std::ios::binary);
                fos.write(uploaded.content.data(),
                          static_cast<std::streamsize>(uploaded.content.size()));
                if (!fos)
                    throw std::runtime_error("failed to write uploaded file");
            }

            const int maxDownloads = parseMaxDownloadsField(req);
            const int port = fileSharer_.offerFile(filePath.string(), displayName, maxDownloads);

            if (server::log::verboseEnabled())
            {
                server::log::info("upload saved: " + filePath.string() + " display=" +
                                  displayName + " invite_port=" + std::to_string(port) +
                                  " max_downloads=" + std::to_string(maxDownloads) +
                                  " bytes=" + std::to_string(uploaded.content.size()));
            }

            const std::string jsonResponse =
                "{\"port\": " + std::to_string(port) +
                ", \"maxDownloads\": " + std::to_string(maxDownloads) + "}";
            res.status = 200;
            res.set_content(jsonResponse, "application/json");
        }
        catch (const std::exception &e)
        {
            server::log::error(std::string("Error processing file upload: ") + e.what());
            res.status = 500;
            res.set_content(std::string("Server error: ") + e.what(), "text/plain");
        }
    }
    bool FileController::parsePortFromPath(const std::string &path, int &outPort)
    {
        const auto slash = path.find_last_of('/');
        if (slash == std::string::npos || slash + 1 >= path.size())
            return false;

        try
        {
            outPort = std::stoi(path.substr(slash + 1));
            return outPort > 0 && outPort <= 65535;
        }
        catch (...)
        {
            return false;
        }
    }

    void FileController::handleDownload(const httplib::Request &req, httplib::Response &res)
    {
        applyCorsHeaders(res);

        if (req.method != "GET")
        {
            res.status = 405;
            res.set_content("Method Not Allowed", "text/plain");
            return;
        }

        int peerPort = 0;
        if (!parsePortFromPath(req.path, peerPort))
        {
            res.status = 400;
            res.set_content("Bad Request: Invalid port number", "text/plain");
            return;
        }

        try
        {
            if (server::log::verboseEnabled())
                server::log::info("download request invite_port=" + std::to_string(peerPort));

            // Claim the invite (sets transferInProgress, rejects if expired/exhausted/busy)
            service::SharedFile shared;
            std::string claimError;
            if (!fileSharer_.claimInviteForTransfer(peerPort, shared, claimError))
            {
                if (claimError.find("invalid") != std::string::npos ||
                    claimError.find("expired") != std::string::npos ||
                    claimError.find("no longer") != std::string::npos ||
                    claimError.find("already in progress") != std::string::npos)
                {
                    res.status = 404;
                    res.set_content(std::string("Not Found: ") + claimError, "text/plain");
                }
                else
                {
                    res.status = 500;
                    res.set_content(std::string("Error downloading file: ") + claimError,
                                    "text/plain");
                }
                return;
            }

            // Verify the file still exists on disk
            if (!fs::exists(shared.path) || !fs::is_regular_file(shared.path))
            {
                fileSharer_.releaseInviteClaim(peerPort);
                res.status = 404;
                res.set_content("Not Found: file no longer on server", "text/plain");
                return;
            }

            const auto fileSize = static_cast<std::size_t>(fs::file_size(shared.path));
            const std::string &filename = shared.downloadName.empty()
                                              ? "downloaded-file"
                                              : shared.downloadName;

            server::log::info("streaming download start invite_port=" +
                              std::to_string(peerPort) + " name=" + filename +
                              " bytes=" + std::to_string(fileSize));

            // Pre-calculate downloads remaining (before this transfer completes)
            const int downloadsRemaining =
                std::max(0, shared.maxDownloads - shared.downloadCount - 1);

            // Set response headers
            setHeaderOnce(res, "Accept-Ranges", "bytes");
            setHeaderOnce(res, "Content-Disposition",
                          "attachment; filename=\"" + filename + "\"");
            setHeaderOnce(res, "X-Filename", filename);
            setHeaderOnce(res, "X-Downloads-Remaining",
                          std::to_string(downloadsRemaining));

            // Capture state for the content provider closure
            auto filePath = std::make_shared<std::string>(shared.path);
            const int invitePort = peerPort;

            // Zero-RAM streaming: httplib calls provider with (offset, length, sink)
            // and handles Range / 206 automatically when content_length is known.
            res.set_content_provider(
                fileSize,
                "application/octet-stream",
                // Provider: read file at requested offset in 64 KB chunks
                [filePath](std::size_t offset, std::size_t length,
                           httplib::DataSink &sink) -> bool
                {
                    std::ifstream file(*filePath, std::ios::binary);
                    if (!file)
                        return false;

                    file.seekg(static_cast<std::streamoff>(offset));
                    if (!file.good())
                        return false;

                    char buffer[65536]; // 64 KB streaming buffer
                    std::size_t remaining = length;
                    while (remaining > 0)
                    {
                        const auto toRead =
                            std::min(sizeof(buffer), remaining);
                        file.read(buffer, static_cast<std::streamsize>(toRead));
                        const auto n =
                            static_cast<std::size_t>(file.gcount());
                        if (n == 0)
                            break;
                        if (!sink.write(buffer, n))
                            return false;
                        remaining -= n;
                    }
                    return true;
                },
                // Releaser: called when response is fully sent or connection drops
                [this, invitePort, filePath](bool success)
                {
                    if (success)
                    {
                        int remaining = 0;
                        fileSharer_.recordSuccessfulDownload(
                            invitePort, *filePath, remaining);
                        server::log::info(
                            "streaming download complete invite_port=" +
                            std::to_string(invitePort) +
                            " remaining=" + std::to_string(remaining));
                    }
                    else
                    {
                        // Connection dropped mid-stream: release claim so client
                        // can retry / resume with a Range header.
                        fileSharer_.releaseInviteClaim(invitePort);
                        server::log::info(
                            "streaming download interrupted invite_port=" +
                            std::to_string(invitePort) +
                            " (claim released for retry/resume)");
                    }
                });

            res.status = 200; // httplib upgrades to 206 if Range header is present
        }
        catch (const std::exception &e)
        {
            server::log::error(std::string("Error downloading file from peer: ") + e.what());
            res.status = 500;
            res.set_content(std::string("Error downloading file: ") + e.what(), "text/plain");
        }
    }

} // namespace server::services