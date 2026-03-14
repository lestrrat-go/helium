<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- Type checking of empty choose branches. The otherwise branch returns a value of the wrong
       type, which results in a type error -->
  
<xsl:strip-space elements="*"/>  
  
<xsl:template match="doc">
<out>
  <xsl:apply-templates select="person[name=('John', 'Jane', 'Prince')]"/>
</out>
</xsl:template>
        
<xsl:template match="person" as="xs:boolean">
  	<xsl:choose>
  	  <xsl:when test="sex='M'">
  	    <xsl:sequence select="true()"/>
      </xsl:when>
  	  <xsl:when test="sex='F'">
  		<xsl:sequence select="false()"/>
      </xsl:when>
  	</xsl:choose>
</xsl:template>

</xsl:stylesheet>
