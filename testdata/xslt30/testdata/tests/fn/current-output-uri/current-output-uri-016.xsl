<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- current-output-uri() used as dynamic function item -->
    
    <xsl:variable name="outUri" select="current-output-uri#0" />    
  
    <xsl:template name="xsl:initial-template">
        <out>current-output-uri-is-empty=<xsl:value-of select="empty($outUri())"/></out>
    </xsl:template>

</xsl:stylesheet>
