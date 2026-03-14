<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- current-output-uri() inside xsl:evaluate, error (XSLT functions in the fn namespace are not allowed) -->
    
    <xsl:param name="xpathExpr"  />
    
    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:evaluate xpath="$xpathExpr" />
        </out>
    </xsl:template>

</xsl:stylesheet>
