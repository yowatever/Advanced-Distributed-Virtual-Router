#include "data_plane.hpp"
#include <iostream>

DataPlane::DataPlane() {}

void DataPlane::add_route(const std::string& destination, const std::string& next_hop, int metric) {
    std::lock_guard<std::mutex> lock(mutex_);
    routes_[destination] = {destination, next_hop, metric};
    std::cout << "DataPlane: Added route " << destination << " -> " << next_hop << std::endl;
}

void DataPlane::delete_route(const std::string& destination) {
    std::lock_guard<std::mutex> lock(mutex_);
    routes_.erase(destination);
    std::cout << "DataPlane: Deleted route " << destination << std::endl;
}

Route DataPlane::get_route(const std::string& destination) const {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = routes_.find(destination);
    if (it != routes_.end()) {
        return it->second;
    }
    return {"", "", -1};
}

std::unordered_map<std::string, Route> DataPlane::get_all_routes() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return routes_;
}
