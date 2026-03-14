<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

    <xsl:template name="main">
        <xsl:variable name="d" select="year-from-date(current-date())"/>
        <out xsl:expand-text="yes">{{The value of {'a}'}</out>
    </xsl:template>
    
</xsl:stylesheet>