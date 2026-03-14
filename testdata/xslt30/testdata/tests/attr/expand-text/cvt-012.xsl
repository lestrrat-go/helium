<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

    <xsl:template name="main" expand-text="yes">
        <xsl:variable name="year" select="year-from-date(current-date())"/>
        <out>{
            $year idiv 1000
        }</out>
    </xsl:template>
    
</xsl:stylesheet>