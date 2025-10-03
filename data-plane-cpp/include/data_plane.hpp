#pragma once

#include <string>
#include <unordered_map>
#include <mutex>

struct Route {
    std::string destination;
    std::string next_hop;
    int metric;
};

class DataPlane {
public:
    DataPlane();
    
    void add_route(const std::string& destination, const std::string& next_hop, int metric);
    void delete_route(const std::string& destination);
    Route get_route(const std::string& destination) const;
    std::unordered_map<std::string, Route> get_all_routes() const;
    
private:
    mutable std::mutex mutex_;
    std::unordered_map<std::string, Route> routes_;
};
