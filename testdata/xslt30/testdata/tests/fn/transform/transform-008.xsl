<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="#all" version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map">
    <xsl:template name="xsl:initial-template">
        <xsl:variable as="map(*)" name="transform-options">
            <xsl:map>
                <xsl:map-entry key="'delivery-format'" select="'raw'" />
                <xsl:map-entry key="'stylesheet-location'">transform-008a.xsl</xsl:map-entry>
                <xsl:map-entry key="'initial-template'"
                    select="QName('http://www.w3.org/1999/XSL/Transform', 'xsl:initial-template')" />
            </xsl:map>
        </xsl:variable>
        <xsl:variable name="result" select="transform($transform-options)"/>
        <result>
            <primary>
                <xsl:sequence select="$result?output"/>
            </primary>
            <secondary>
                <xsl:sequence select="$result('http://www.example.com/out.xml')"/>
            </secondary>
        </result>
    </xsl:template>
</xsl:stylesheet>

