<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:sf="http://streamable-functions"
    version="3.0">
    
    <xsl:mode streamable="yes" />
    
    <xsl:template match="/">
        <xsl:copy-of select="*/section[1][sf:ascent(.)/local-name() = 'doc']/section/p[2]" />
    </xsl:template>
    
    <xsl:function name="sf:ascent" streamability="ascent" as="node()*">
        <xsl:param name="node" as="node()"/>
        <xsl:sequence select="$node/ancestor-or-self::*[last()]" />
    </xsl:function>
</xsl:stylesheet>