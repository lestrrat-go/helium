<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    expand-text="no"
    version="3.0">

    <xsl:template name="main" expand-text="TRUE">
        <out>
            <xsl:attribute name="a">{1,2} {3,4}{5,6}</xsl:attribute>
            <xsl:attribute name="b" separator=" ">{1,2} {3,4}{5,6}</xsl:attribute>
        </out>
    </xsl:template>
    
</xsl:stylesheet>