<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:mf="http://example.com/mf"
    exclude-result-prefixes="xs map mf" version="3.0">
    
    <!-- Map key must not be an empty sequence. Saxon bug #5414 -->
    
    <xsl:param name="p" select="1 to 20"/>

    <xsl:template match="/" name="xsl:initial-template">
        <xsl:variable name="map" as="map(*)">
            <xsl:map>
                <xsl:map-entry key="$p[. lt 0]" select="42"/>
            </xsl:map>
        </xsl:variable>
        <out size="{map:size($map)}"/>
    </xsl:template>

</xsl:stylesheet>
