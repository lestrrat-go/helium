<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    expand-text="no"
    version="3.0">
    
    <xsl:param name="Hello">Goodbye</xsl:param>
    <xsl:param name="World">Mars</xsl:param>
    
    <xsl:template name="main" expand-text=" yes ">
        <out>{$Hello} {$World}!</out>
    </xsl:template>
    
</xsl:stylesheet>