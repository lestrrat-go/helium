<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:sf="http://streamable-functions"
    version="3.0">
    
    <xsl:mode streamable="yes" />
    
    <xsl:template match="/">
        <xsl:if test="sf:inspection(.)" >
            <xsl:copy-of select="doc/section[position() = 1 and sf:inspection(.) = false()]/section/p[2][sf:inspection(.) = false()]" />
        </xsl:if>
    </xsl:template>
    
    <xsl:function name="sf:inspection" streamability="inspection" as="xs:boolean">
        <xsl:param name="node" as="node()"/>
        <xsl:sequence select="$node instance of document-node()" />
    </xsl:function>
</xsl:stylesheet>