<?spec xslt#grouping?>
<xsl:transform 
  version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- Use group-adjacent to merge a set of adjacent bullets under an envelope element -->
  
  <xsl:template match="/">
    <o><xsl:apply-templates/></o>
  </xsl:template>  
  
<xsl:template match="p">
    <xsl:for-each-group select="node()" 
            group-adjacent="self::ul or self::ol">
        <xsl:choose>
            <xsl:when test="self::ul or self::ol">
                <xsl:copy-of select="current-group()"/>  
            </xsl:when>
            <xsl:otherwise>
                <p>
                    <xsl:copy-of select="current-group()"/>
                </p>
            </xsl:otherwise>  
        </xsl:choose>
    </xsl:for-each-group>
</xsl:template>    
	
</xsl:transform>