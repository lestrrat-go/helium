<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

    <xsl:template name="main">
        <xsl:variable name="d" as="attribute()">
          <xsl:attribute name="i" type="xs:integer">0023</xsl:attribute>
        </xsl:variable>
        <out xsl:expand-text="yes">{$d}</out>
    </xsl:template>
    
</xsl:stylesheet>