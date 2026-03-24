<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:fo="http://www.w3.org/1999/XSL/Format"
    package-version="1.0"
    name="urn:170b"
    exclude-result-prefixes="xs"
    version="3.0">
    
    <xsl:mode/>
    
    <xsl:use-package name="urn:170c" package-version="1.0">
        <xsl:accept component="template" names="*" visibility="public" />
        <xsl:accept component="function" names="*:setRed" visibility="public" />
        <xsl:accept component="mode" names="normal" visibility="public" />
        <xsl:override>
            <xsl:template match="*[contains(@class,' topic/ph ')]" mode="normal">
                <fo:inline>
                    <xsl:text>ph:</xsl:text>
                    <xsl:apply-templates/>
                </fo:inline>
            </xsl:template>
        </xsl:override>
    </xsl:use-package>
    
</xsl:package>
