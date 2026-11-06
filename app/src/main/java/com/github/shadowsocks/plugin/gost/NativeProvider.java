package com.github.shadowsocks.plugin.gost;

import android.net.Uri;
import android.os.ParcelFileDescriptor;
import com.github.shadowsocks.plugin.NativePluginProvider;
import com.github.shadowsocks.plugin.PathProvider;

import org.jetbrains.annotations.NonNls;
import org.jetbrains.annotations.NotNull;
import org.jetbrains.annotations.Nullable;

import java.io.File;
import java.io.FileNotFoundException;

public class NativeProvider extends NativePluginProvider {

    @NonNls
    public static final String NATIVE_PLUGIN_NAME = "libgost-plugin.so";
    @NonNls
    public static final String NATIVE_PATH = "gost-antidpi";

    @NotNull
    @Override
    public ParcelFileDescriptor openFile(@Nullable Uri uri) {
        try {
            String executable = uri != null && "libbyedpi.so".equals(uri.getLastPathSegment())
                    ? getContext().getApplicationInfo().nativeLibraryDir + "/libbyedpi.so"
                    : getExecutable();
            return ParcelFileDescriptor.open(new File(executable), ParcelFileDescriptor.MODE_READ_ONLY);
        } catch (FileNotFoundException e) {
            throw new RuntimeException(e);
        }
    }

    @NotNull
    @Override
    public String getExecutable() {
        return getContext().getApplicationInfo().nativeLibraryDir + "/" + NATIVE_PLUGIN_NAME;
    }

    @Override
    protected void populateFiles(@NotNull PathProvider provider) {
        provider.addPath(NATIVE_PATH, 0755);
        provider.addPath("libbyedpi.so", 0755);
    }
}
