<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="2.0">
    <!-- Test the empty instruction xsl:value, which should raise an error in the XSLT 2.0 specification. -->
    <xsl:output method="xml" indent="no"/>
    
    <xsl:template match="/">
        <out>
            <xsl:value-of />
        </out>
    </xsl:template>
</xsl:stylesheet>