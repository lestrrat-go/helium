<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="3.0">
    <xsl:param name="node-name" required="yes"/>
    <xsl:mode streamable="yes"/>
    <xsl:template name="main">
      <xsl:source-document streamable="yes" href="si-for-each-801.xml">
        <root>
            <xsl:for-each select="//node()[name() = $node-name]">
                <found/>
            </xsl:for-each>
        </root>
      </xsl:source-document>  
    </xsl:template>
</xsl:stylesheet>