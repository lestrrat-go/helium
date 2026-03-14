<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:sf="http://streamable-functions"
    version="3.0">
    
    <xsl:mode streamable="yes" />
    
    <xsl:template match="/">
        <xsl:copy-of select="sf:absorbing(.)" />
    </xsl:template>
    
    <xsl:function name="sf:absorbing" streamability="absorbing">
        <xsl:param name="node" as="node()"/>
        <xsl:copy-of select="$node/doc/section[1]/section/p[2]" />
    </xsl:function>
</xsl:stylesheet>