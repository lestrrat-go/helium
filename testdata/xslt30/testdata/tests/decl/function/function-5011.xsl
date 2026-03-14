<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:sf="http://streamable-functions"
    version="3.0">
    
    <xsl:mode streamable="yes" />
    
    <xsl:template match="/">
        <xsl:copy-of select="(doc/section/sf:unclassified(copy-of(.))[self::new]/p)[1]" />
    </xsl:template>
    
    <xsl:function name="sf:unclassified" streamability="unclassified" as="node()">
        <xsl:param name="node" />
        <new>
            <xsl:copy-of select="$node/section/p[2]" />
        </new>
    </xsl:function>
</xsl:stylesheet>