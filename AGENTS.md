# Quy tắc phát triển & Quản lý Git (Workspace Rules)

## 1. Kiểm soát quyền Git Push (BẮT BUỘC)
- **YÊU CẦU XÁC NHẬN TRƯỚC KHI PUSH:** Trước khi thực hiện lệnh `git push` (đẩy code lên GitHub hoặc bất kỳ remote repository nào), trợ lý AI **BẮT BUỘC PHẢI DỪNG LẠI**, thông báo tóm tắt các commit/thay đổi và **HỎI Ý KIẾN XÁC NHẬN (CONFIRM)** từ người dùng.
- **TUYỆT ĐỐI KHÔNG TỰ Ý CHẠY `git push`:** Chỉ được phép thực hiện `git push` khi người dùng đã phản hồi đồng ý rõ ràng (ví dụ: "đồng ý", "push đi", "ok", "xác nhận"). 
- Nếu người dùng chưa xác nhận hoặc yêu cầu xem lại, giữ nguyên code ở local branch.

