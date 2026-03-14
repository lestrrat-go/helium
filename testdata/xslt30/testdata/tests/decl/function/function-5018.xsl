<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:sf="http://streamable-functions"
    exclude-result-prefixes="xs sf"
    version="3.0">
    
    <!-- inlined equivalent of 5017, for comparison -->
    
    <xsl:mode streamable="yes" />
    
    <xsl:template match="/">
        <result>
            <xsl:for-each select="doc//section//p/ancestor-or-self::section">
            <inverted-section name="{@index}" depth="{count(ancestor-or-self::section)}" />
        </xsl:for-each>
        </result>
    </xsl:template>
    
    <!-- returns overlapping ascending nodes -->
    <!--<xsl:function name="sf:ascent" streamability="ascent" as="node()*">
        <xsl:param name="node" as="node()"/>
        <xsl:sequence select="$node/ancestor-or-self::section" />
    </xsl:function>-->
</xsl:stylesheet>