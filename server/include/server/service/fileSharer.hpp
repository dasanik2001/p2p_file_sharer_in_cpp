#pragma once

#include <atomic>
#include <chrono>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <unordered_map>

namespace server::service
{

    struct SharedFile
    {
        std::string path;
        std::string downloadName; // original basename sent to clients (keeps extension)
        int maxDownloads{1};
        int downloadCount{0};
        bool transferInProgress{false};
        std::chrono::steady_clock::time_point offeredAt{};
    };

    class FileSharer
    {
    public:
        FileSharer();
        ~FileSharer();

        FileSharer(const FileSharer &) = delete;
        FileSharer &operator=(const FileSharer &) = delete;

        void shutdown();

        int offerFile(const std::string &filePath, const std::string &downloadName,
                      int maxDownloads = 1);

        /** Legacy: accept one remote/local connection (not used on upload path). */
        void startFileServer(int port);

        bool tryGetDownloadName(int port, std::string &outName) const;
        bool tryGetSharedFile(int port, SharedFile &outFile) const;

        /**
         * HTTP download: P2P over socketpair; invite removed after maxDownloads successes.
         */
        bool receiveViaLocalP2P(int port, std::string &outFilename, std::string &outBody,
                                std::string &outError, int *outDownloadsRemaining = nullptr);

        /**
         * Streaming download helpers — used by FileController with httplib content providers.
         * claimInviteForTransfer: marks invite in-progress, rejects if expired/exhausted/busy.
         * releaseInviteClaim: clears in-progress flag without consuming a download slot.
         * recordSuccessfulDownload: increments count; removes invite when quota is reached.
         */
        bool claimInviteForTransfer(int port, SharedFile &outFile, std::string &outError);
        void releaseInviteClaim(int port);
        void recordSuccessfulDownload(int port, const std::string &filePath,
                                      int &outDownloadsRemaining);

    private:
        mutable std::mutex mapMutex_;
        std::unordered_map<int, SharedFile> available_files_;
        std::unordered_map<int, std::unique_ptr<std::mutex>> portTransferMutexes_;

        std::thread expirySweeperThread_;
        std::atomic<bool> stopExpirySweeper_{false};
        std::atomic<bool> expirySweeperStarted_{false};

        static std::chrono::seconds inviteTtl();
        static std::chrono::seconds expirySweepInterval();

        void ensureExpirySweeperStarted();
        void expirySweeperLoop();
        void purgeExpiredInvites();
        bool isInviteExpired(const SharedFile &file) const;

        std::mutex &transferMutexForPort(int port);
        void consumeInvite(int port, const std::string &filePath);
        void expireIdleInvite(int port, const std::string &filePath);
    };

} // namespace server::service

