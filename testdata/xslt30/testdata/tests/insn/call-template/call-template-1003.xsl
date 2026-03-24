<?xml version="1.0"?> 
<?spec xslt#named-templates?>
<xsl:stylesheet 
xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" expand-text="yes">
<xsl:output method="xml"/>

<!-- test tail recursion within a singleton xsl:for-each -->
   
   <xsl:variable name="in">
      <doc>
         <xsl:for-each select="1 to 1000">
            <e>{.}</e>
         </xsl:for-each>
      </doc>
   </xsl:variable>

  <xsl:template name="xsl:initial-template">
  	<out>
  	   <xsl:apply-templates select="$in/doc/e[1]"/>
  	</out>
  </xsl:template>
   
   <xsl:template match="e">
      <xsl:if test=". mod 100 = 1">{.} </xsl:if>
      <xsl:for-each select="following-sibling::e[1]">
         <xsl:apply-templates select="."/>
      </xsl:for-each>     
   </xsl:template>


</xsl:stylesheet>
