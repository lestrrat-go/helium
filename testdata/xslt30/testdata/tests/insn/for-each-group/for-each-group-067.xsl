<?spec xslt#grouping?>
<xsl:stylesheet
   version="2.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<!-- an interesting use case for group-adjacent, used to identify consecutive
     items that are not equal, but contiguous in value -->

<xsl:template match="/"><z><xsl:next-match/></z></xsl:template>

<xsl:template match="r">
<r>
<xsl:for-each-group select="x" 
         group-adjacent="
           if (position()=1 or 
            .=preceding-sibling::x[1]+1) 
           then -1 
           else position() "> 
  <xsl:value-of select="."/> 
  <xsl:if test="current-group()[2]"> 
    <xsl:text>-</xsl:text> 
    <xsl:value-of select="current-group()[last()]"/>
  </xsl:if>
  <xsl:if test="position()!=last()">,</xsl:if>
</xsl:for-each-group>
</r>
</xsl:template>



</xsl:stylesheet>

