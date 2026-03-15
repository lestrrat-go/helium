<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- tests a Saxon 8.3 bug in optimization of empty choose branches -->
  
<xsl:strip-space elements="*"/>  
  
<xsl:template match="doc">
<out>
  <xsl:apply-templates/>
</out>
</xsl:template>
        
<xsl:template match="person">
  	<xsl:choose>
  	  <xsl:when test="sex='M'">
  	    <male/>
  		</xsl:when>
  		<xsl:when test="sex='F'">
  		</xsl:when>
  		<xsl:otherwise>
  		</xsl:otherwise>
  	</xsl:choose>
</xsl:template>

</xsl:stylesheet>
