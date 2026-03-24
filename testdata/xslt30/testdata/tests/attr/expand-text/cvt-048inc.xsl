<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    expand-text="yes"
    version="3.0">
    
    <xsl:template name="sub" match="/never">
        <out>{$Hello} {$World}!</out>
    </xsl:template>
    
</xsl:stylesheet>