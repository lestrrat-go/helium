<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    name="urn:simple"
    package-version="1.0-alpha"
    version="3.0">
    
    <xsl:output omit-xml-declaration="yes" />
    
    <xsl:template name="xsl:initial-template" visibility="public">
        <package-version>
            <xsl:value-of select="doc('')//@package-version" />
        </package-version>
    </xsl:template>
    
</xsl:package>