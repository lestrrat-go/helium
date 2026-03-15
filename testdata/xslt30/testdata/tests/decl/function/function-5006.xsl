<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:sf="http://streamable-functions"
    exclude-result-prefixes="xs sf"
    version="3.0">
    
    <xsl:mode streamable="yes" />
    
    <xsl:template match="/">
        <!-- scanning expression with posture striding -->
        <result>
            <xsl:for-each select="outermost(doc/sf:deep-descent(section[1]))/text()" >
                <xsl:copy-of select="self::text()" />
            </xsl:for-each>
        </result>
    </xsl:template>
    
    <xsl:function name="sf:deep-descent"  as="node()*" streamability="deep-descent">
        <xsl:param name="node" as="node()"/>
        <!-- Written this way to circumvent a bug in the streamability rules: $node//section/p is not a scanning expression,
            because when considered as a pattern it is not motionless, because a pattern that contains a rooted path (like $node)
            is deemed not to be motionless. See bug 30064. -->
        <xsl:for-each select="$node">
            <xsl:sequence select="self::node()//section//p" />    
        </xsl:for-each>        
    </xsl:function>
</xsl:stylesheet>