type GoogleCredentialHandler = (response: google.accounts.id.CredentialResponse) => void;

type GoogleSignInOptions = {
    prompt?: boolean;
    renderButton?: boolean;
};

export function initializeGoogleSignIn(
    buttonEl: HTMLDivElement | null,
    clientId: string,
    onCredential: GoogleCredentialHandler,
    options: GoogleSignInOptions = {}
): boolean {
    const googleApi = window.google;
    if (!googleApi?.accounts?.id) {
        return false;
    }

    googleApi.accounts.id.initialize({
        client_id: clientId,
        callback: onCredential,
        auto_select: false,
        // TODO: Re-enable FedCM once Google origin/client ID config is fixed.
        // Current error: "The given origin is not allowed for the given client ID."
        use_fedcm_for_prompt: false,
    });

    const shouldRenderButton = options.renderButton ?? true;
    if (buttonEl && shouldRenderButton) {
        googleApi.accounts.id.renderButton(buttonEl, {
            type: "standard",
            theme: "outline",
            size: "large",
            text: "signin_with",
            shape: "pill",
        });
    }

    if (options.prompt ?? true) {
        googleApi.accounts.id.prompt();
    }
    return true;
}

export function cancelGoogleSignIn() {
    window.google?.accounts?.id?.cancel?.();
}
