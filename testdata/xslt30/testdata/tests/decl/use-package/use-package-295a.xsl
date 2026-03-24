<xsl:stylesheet version="3.0" exclude-result-prefixes="#all"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:local="http://localhost/pkg"
    expand-text="yes">
    
    <xsl:use-package name="http://use-package-295b/pkg"/>
    
    <xsl:template name="xsl:initial-template">
        hello {local:bar()}
    </xsl:template>
    
</xsl:stylesheet>